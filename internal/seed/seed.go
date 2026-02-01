package seed

import (
	"bookstore/internal/models"
	"bookstore/internal/store"

	"golang.org/x/crypto/bcrypt"
)

func MustSeed(s *store.Store) {
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	_, _ = s.CreateUser(models.User{
		Username:     "admin",
		Email:        "admin@bookstore.local",
		PasswordHash: string(adminHash),
		Role:         "Admin",
	})

	b1, _ := s.CreateBook(models.Book{Title: "Go in Action", Author: "William Kennedy", Description: "Practical Go book"})
	b2, _ := s.CreateBook(models.Book{Title: "Clean Code", Author: "Robert C. Martin", Description: "Code quality principles"})

	_, _ = s.CreateFormat(models.BookFormat{BookID: b1.ID, Type: "Physical", Price: 12000, StockQuantity: 10})
	_, _ = s.CreateFormat(models.BookFormat{BookID: b1.ID, Type: "Digital", Price: 7000, StockQuantity: 9999})
	_, _ = s.CreateFormat(models.BookFormat{BookID: b1.ID, Type: "Audio", Price: 9000, StockQuantity: 9999})

	_, _ = s.CreateFormat(models.BookFormat{BookID: b2.ID, Type: "Physical", Price: 15000, StockQuantity: 7})
	_, _ = s.CreateFormat(models.BookFormat{BookID: b2.ID, Type: "Digital", Price: 8000, StockQuantity: 9999})
}
