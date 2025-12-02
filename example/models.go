package main

import "time"

// User represents a user in the system
// Implements orm.TableStruct interface
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Age       int       `json:"age" db:"age"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TableName implements the orm.TableStruct interface
func (u *User) TableName() string {
	return "users"
}

// Product represents a product in the catalog
// Implements orm.TableStruct interface
type Product struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Price     float64   `json:"price" db:"price"`
	Stock     int       `json:"stock" db:"stock"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TableName implements the orm.TableStruct interface
func (p *Product) TableName() string {
	return "products"
}

// Order represents a customer order
// Implements orm.TableStruct interface
type Order struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Total     float64   `json:"total" db:"total"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TableName implements the orm.TableStruct interface
func (o *Order) TableName() string {
	return "orders"
}

// OrderItem represents a line item in an order
// Implements orm.TableStruct interface
type OrderItem struct {
	ID        int     `json:"id" db:"id"`
	OrderID   int     `json:"order_id" db:"order_id"`
	ProductID int     `json:"product_id" db:"product_id"`
	Quantity  int     `json:"quantity" db:"quantity"`
	Price     float64 `json:"price" db:"price"`
}

// TableName implements the orm.TableStruct interface
func (oi *OrderItem) TableName() string {
	return "order_items"
}
