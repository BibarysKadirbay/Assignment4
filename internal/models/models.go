package models

import "time"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type Book struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

type BookFormat struct {
	ID            int     `json:"id"`
	BookID        int     `json:"book_id"`
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	StockQuantity int     `json:"stock_quantity"`
}

type Order struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	OrderDate   time.Time `json:"order_date"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
}

type OrderItem struct {
	ID              int     `json:"id"`
	OrderID         int     `json:"order_id"`
	FormatID        int     `json:"format_id"`
	Quantity        int     `json:"quantity"`
	PriceAtPurchase float64 `json:"price_at_purchase"`
}

type DigitalAccess struct {
	ID                int       `json:"id"`
	UserID            int       `json:"user_id"`
	FormatID          int       `json:"format_id"`
	AccessGrantedDate time.Time `json:"access_granted_date"`
}

type BookWithFormats struct {
	Book    Book         `json:"book"`
	Formats []BookFormat `json:"formats"`
}

type LibraryItem struct {
	Access DigitalAccess `json:"access"`
	Book   Book          `json:"book"`
	Format BookFormat    `json:"format"`
}
