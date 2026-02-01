package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bookstore/internal/models"
	"bookstore/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type API struct {
	Store      *store.Store
	OrderQueue chan<- int
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func mapErr(err error) (int, any) {
	if err == nil {
		return http.StatusOK, nil
	}
	switch {
	case errors.Is(err, store.ErrInvalidInput):
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, map[string]any{"error": err.Error()}
	case errors.Is(err, store.ErrConflict):
		return http.StatusConflict, map[string]any{"error": err.Error()}
	case errors.Is(err, store.ErrInsufficientStock):
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	default:
		return http.StatusInternalServerError, map[string]any{"error": "internal error"}
	}
}

func (a *API) RegisterRoutes(r *Router) {
	r.HandleFunc("/health", a.handleHealth)

	r.HandleFunc("/users/register", a.handleRegister)
	r.HandleFunc("/users/login", a.handleLogin)

	r.HandleFunc("/books", a.handleBooks)
	r.HandleFunc("/books/", a.handleBookByID)

	r.HandleFunc("/orders", a.handleOrders)
	r.HandleFunc("/orders/", a.handleOrderByID)

	r.HandleFunc("/users/", a.handleUsersSubroutes)

	r.HandleFunc("/admin/books", a.handleAdminBooks)
	r.HandleFunc("/admin/books/", a.handleAdminBooksByID)
	r.HandleFunc("/admin/formats/", a.handleAdminFormatsByID)
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal error"})
		return
	}
	u, err := a.Store.CreateUser(models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
	})
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	u, err := a.Store.FindUserByEmail(req.Email)
	if err != nil {
		code, body := mapErr(store.ErrNotFound)
		writeJSON(w, code, body)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":  u,
		"token": strconv.Itoa(u.ID),
	})
}

func (a *API) handleBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	search := r.URL.Query().Get("search")
	books, err := a.Store.ListBooks(search)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, books)
}

func (a *API) handleBookByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	idStr, ok := PathParamAfter("/books", r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}
	b, err := a.Store.GetBook(id)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	list, _ := a.Store.ListBooks("")
	var formats []models.BookFormat
	for _, bw := range list {
		if bw.Book.ID == b.ID {
			formats = bw.Formats
			break
		}
	}
	writeJSON(w, http.StatusOK, models.BookWithFormats{Book: b, Formats: formats})
}

func (a *API) handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req store.CreateOrderRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	owi, err := a.Store.CreateOrder(req)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	select {
	case a.OrderQueue <- owi.Order.ID:
	default:
	}
	writeJSON(w, http.StatusCreated, owi)
}

func (a *API) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	idStr, ok := PathParamAfter("/orders", r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/cancel") {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		var body struct {
			UserID int `json:"user_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		o, err := a.Store.CancelOrder(id, body.UserID)
		if err != nil {
			code, b := mapErr(err)
			writeJSON(w, code, b)
			return
		}
		writeJSON(w, http.StatusOK, o)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	o, err := a.Store.GetOrder(id)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (a *API) handleUsersSubroutes(w http.ResponseWriter, r *http.Request) {
	idStr, ok := PathParamAfter("/users", r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/orders") {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		orders, err := a.Store.ListOrdersByUser(id)
		if err != nil {
			code, body := mapErr(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, orders)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/library") {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		lib, err := a.Store.ListLibrary(id)
		if err != nil {
			code, body := mapErr(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, lib)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	u, err := a.Store.GetUser(id)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *API) handleAdminBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var b models.Book
	if err := readJSON(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	created, err := a.Store.CreateBook(b)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *API) handleAdminBooksByID(w http.ResponseWriter, r *http.Request) {
	idStr, ok := PathParamAfter("/admin/books", r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}

	if strings.Contains(r.URL.Path, "/formats") {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		var req models.BookFormat
		if err := readJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		req.BookID = id
		f, err := a.Store.CreateFormat(req)
		if err != nil {
			code, body := mapErr(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusCreated, f)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var patch models.Book
		if err := readJSON(r, &patch); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		updated, err := a.Store.UpdateBook(id, patch)
		if err != nil {
			code, body := mapErr(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		err := a.Store.DeleteBook(id)
		if err != nil {
			code, body := mapErr(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (a *API) handleAdminFormatsByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	idStr, ok := PathParamAfter("/admin/formats", r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}
	var patch models.BookFormat
	if err := readJSON(r, &patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	updated, err := a.Store.UpdateFormat(id, patch)
	if err != nil {
		code, body := mapErr(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
