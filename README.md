# Online Bookstore Management System (Milestone 2)

This is a minimal backend that follows the Assignment 3 ERD and satisfies:
- net/http server with JSON input/output
- Core domain models: User, Book, BookFormat, Order, OrderItem, DigitalAccess
- 3+ core features: register/login, book catalog/search/details, create/view/cancel orders, digital library, admin book CRUD + format management
- In-memory persistence with safe concurrent access (RWMutex)
- Concurrency: background order processor goroutine using a channel

## Run
```bash
go mod tidy
go run ./cmd/server
```

Server listens on `:8080`.

## Quick Demo (Postman or curl)

### 1) Register user
POST /users/register
```json
{"username":"alice","email":"alice@mail.com","password":"pass123","role":"Customer"}
```

### 2) Login
POST /users/login
```json
{"email":"alice@mail.com","password":"pass123"}
```

### 3) List books (search)
GET /books?search=go

### 4) Create order
POST /orders
```json
{"user_id":1,"items":[{"format_id":1,"quantity":1}]}
```
Order is created as `Pending`, then background worker marks it `Completed` and grants digital access for Digital/Audio formats.

### 5) View orders
GET /users/1/orders

### 6) View digital library
GET /users/1/library

### Admin examples
- POST /admin/books
- PUT /admin/books/{id}
- DELETE /admin/books/{id}
- POST /admin/books/{id}/formats
- PUT /admin/formats/{id}
