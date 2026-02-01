package store

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bookstore/internal/models"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type Store struct {
	mu sync.RWMutex

	userSeq   uint64
	bookSeq   uint64
	formatSeq uint64
	orderSeq  uint64
	itemSeq   uint64
	accessSeq uint64

	users         map[int]models.User
	books         map[int]models.Book
	formats       map[int]models.BookFormat
	orders        map[int]models.Order
	orderItems    map[int]models.OrderItem
	digitalAccess map[int]models.DigitalAccess
}

func New() *Store {
	return &Store{
		users:         make(map[int]models.User),
		books:         make(map[int]models.Book),
		formats:       make(map[int]models.BookFormat),
		orders:        make(map[int]models.Order),
		orderItems:    make(map[int]models.OrderItem),
		digitalAccess: make(map[int]models.DigitalAccess),
	}
}

func (s *Store) nextID(seq *uint64) int {
	return int(atomic.AddUint64(seq, 1))
}

func (s *Store) CreateUser(u models.User) (models.User, error) {
	if u.Username == "" || u.Email == "" || u.PasswordHash == "" {
		return models.User{}, ErrInvalidInput
	}
	if u.Role == "" {
		u.Role = "Customer"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return models.User{}, ErrConflict
		}
	}
	u.ID = s.nextID(&s.userSeq)
	s.users[u.ID] = u
	return u, nil
}

func (s *Store) GetUser(id int) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return models.User{}, ErrNotFound
	}
	return u, nil
}

func (s *Store) FindUserByEmail(email string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return models.User{}, ErrNotFound
}

func (s *Store) CreateBook(b models.Book) (models.Book, error) {
	if b.Title == "" || b.Author == "" {
		return models.Book{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b.ID = s.nextID(&s.bookSeq)
	s.books[b.ID] = b
	return b, nil
}

func (s *Store) UpdateBook(id int, patch models.Book) (models.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.books[id]
	if !ok {
		return models.Book{}, ErrNotFound
	}
	if patch.Title != "" {
		b.Title = patch.Title
	}
	if patch.Author != "" {
		b.Author = patch.Author
	}
	if patch.Description != "" {
		b.Description = patch.Description
	}
	s.books[id] = b
	return b, nil
}

func (s *Store) DeleteBook(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.books[id]; !ok {
		return ErrNotFound
	}
	delete(s.books, id)
	for fid, f := range s.formats {
		if f.BookID == id {
			delete(s.formats, fid)
		}
	}
	return nil
}

func (s *Store) GetBook(id int) (models.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.books[id]
	if !ok {
		return models.Book{}, ErrNotFound
	}
	return b, nil
}

func (s *Store) ListBooks(search string) ([]models.BookWithFormats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	search = strings.TrimSpace(strings.ToLower(search))
	out := make([]models.BookWithFormats, 0, len(s.books))
	for _, b := range s.books {
		if search != "" {
			if !strings.Contains(strings.ToLower(b.Title), search) && !strings.Contains(strings.ToLower(b.Author), search) {
				continue
			}
		}
		bw := models.BookWithFormats{Book: b}
		for _, f := range s.formats {
			if f.BookID == b.ID {
				bw.Formats = append(bw.Formats, f)
			}
		}
		out = append(out, bw)
	}
	return out, nil
}

func (s *Store) CreateFormat(f models.BookFormat) (models.BookFormat, error) {
	if f.BookID <= 0 || f.Type == "" || f.Price < 0 || f.StockQuantity < 0 {
		return models.BookFormat{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.books[f.BookID]; !ok {
		return models.BookFormat{}, ErrNotFound
	}
	f.ID = s.nextID(&s.formatSeq)
	s.formats[f.ID] = f
	return f, nil
}

func (s *Store) UpdateFormat(id int, patch models.BookFormat) (models.BookFormat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.formats[id]
	if !ok {
		return models.BookFormat{}, ErrNotFound
	}
	if patch.Type != "" {
		f.Type = patch.Type
	}
	if patch.Price != 0 {
		f.Price = patch.Price
	}
	if patch.StockQuantity != 0 {
		f.StockQuantity = patch.StockQuantity
	}
	s.formats[id] = f
	return f, nil
}

func (s *Store) GetFormat(id int) (models.BookFormat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.formats[id]
	if !ok {
		return models.BookFormat{}, ErrNotFound
	}
	return f, nil
}

type CreateOrderItem struct {
	FormatID int `json:"format_id"`
	Quantity int `json:"quantity"`
}

type CreateOrderRequest struct {
	UserID int               `json:"user_id"`
	Items  []CreateOrderItem `json:"items"`
}

type OrderWithItems struct {
	Order models.Order       `json:"order"`
	Items []models.OrderItem `json:"items"`
}

func (s *Store) CreateOrder(req CreateOrderRequest) (OrderWithItems, error) {
	if req.UserID <= 0 || len(req.Items) == 0 {
		return OrderWithItems{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[req.UserID]; !ok {
		return OrderWithItems{}, ErrNotFound
	}

	for _, it := range req.Items {
		if it.FormatID <= 0 || it.Quantity <= 0 {
			return OrderWithItems{}, ErrInvalidInput
		}
		f, ok := s.formats[it.FormatID]
		if !ok {
			return OrderWithItems{}, ErrNotFound
		}
		if f.StockQuantity < it.Quantity {
			return OrderWithItems{}, ErrInsufficientStock
		}
	}

	order := models.Order{
		ID:        s.nextID(&s.orderSeq),
		UserID:    req.UserID,
		OrderDate: time.Now().UTC(),
		Status:    "Pending",
	}
	var total float64
	items := make([]models.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		f := s.formats[it.FormatID]
		f.StockQuantity -= it.Quantity
		s.formats[it.FormatID] = f
		oi := models.OrderItem{
			ID:              s.nextID(&s.itemSeq),
			OrderID:         order.ID,
			FormatID:        it.FormatID,
			Quantity:        it.Quantity,
			PriceAtPurchase: f.Price,
		}
		s.orderItems[oi.ID] = oi
		items = append(items, oi)
		total += float64(it.Quantity) * f.Price
	}
	order.TotalAmount = total
	s.orders[order.ID] = order
	return OrderWithItems{Order: order, Items: items}, nil
}

func (s *Store) GetOrder(id int) (models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok {
		return models.Order{}, ErrNotFound
	}
	return o, nil
}

func (s *Store) ListOrdersByUser(userID int) ([]OrderWithItems, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OrderWithItems, 0)
	for _, o := range s.orders {
		if o.UserID != userID {
			continue
		}
		owi := OrderWithItems{Order: o}
		for _, it := range s.orderItems {
			if it.OrderID == o.ID {
				owi.Items = append(owi.Items, it)
			}
		}
		out = append(out, owi)
	}
	return out, nil
}

func (s *Store) CancelOrder(orderID int, userID int) (models.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok {
		return models.Order{}, ErrNotFound
	}
	if o.UserID != userID {
		return models.Order{}, ErrInvalidInput
	}
	if o.Status != "Pending" {
		return models.Order{}, ErrConflict
	}
	for _, it := range s.orderItems {
		if it.OrderID == orderID {
			f := s.formats[it.FormatID]
			f.StockQuantity += it.Quantity
			s.formats[it.FormatID] = f
		}
	}
	o.Status = "Cancelled"
	s.orders[orderID] = o
	return o, nil
}

func (s *Store) MarkOrderCompleted(orderID int) (models.Order, []models.OrderItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok {
		return models.Order{}, nil, ErrNotFound
	}
	if o.Status != "Pending" {
		return o, nil, ErrConflict
	}
	o.Status = "Completed"
	s.orders[orderID] = o

	items := make([]models.OrderItem, 0)
	for _, it := range s.orderItems {
		if it.OrderID == orderID {
			items = append(items, it)
		}
	}
	return o, items, nil
}

func (s *Store) GrantDigitalAccess(userID int, formatID int) (models.DigitalAccess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return models.DigitalAccess{}, ErrNotFound
	}
	f, ok := s.formats[formatID]
	if !ok {
		return models.DigitalAccess{}, ErrNotFound
	}
	if strings.EqualFold(f.Type, "Physical") {
		return models.DigitalAccess{}, ErrConflict
	}
	for _, a := range s.digitalAccess {
		if a.UserID == userID && a.FormatID == formatID {
			return a, nil
		}
	}
	a := models.DigitalAccess{
		ID:                s.nextID(&s.accessSeq),
		UserID:            userID,
		FormatID:          formatID,
		AccessGrantedDate: time.Now().UTC(),
	}
	s.digitalAccess[a.ID] = a
	return a, nil
}

func (s *Store) ListLibrary(userID int) ([]models.LibraryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.users[userID]; !ok {
		return nil, ErrNotFound
	}
	out := make([]models.LibraryItem, 0)
	for _, a := range s.digitalAccess {
		if a.UserID != userID {
			continue
		}
		f, ok := s.formats[a.FormatID]
		if !ok {
			continue
		}
		b, ok := s.books[f.BookID]
		if !ok {
			continue
		}
		out = append(out, models.LibraryItem{
			Access: a,
			Book:   b,
			Format: f,
		})
	}
	return out, nil
}
