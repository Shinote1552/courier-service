package entity

import "time"

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusCooking    OrderStatus = "cooking"
	OrderStatusDelivering OrderStatus = "delivering"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCanceled   OrderStatus = "canceled"
	OrderStatusDeleted    OrderStatus = "deleted"
	OrderStatusCreated    OrderStatus = "created"
	OrderStatusUpdated    OrderStatus = "updated"
	OrderStatusCompleted  OrderStatus = "completed"
)

type Item struct {
	FoodID   string
	Name     string
	Quantity int
	Price    int
}

type DeliveryAddress struct {
	Street    string
	House     string
	Apartment string
	Floor     string
	Comment   string
}

type Order struct {
	ID                string
	UserID            string
	OrderNumber       string
	FIO               string
	RestaurantID      string
	Items             []Item
	TotalPrice        int64
	Address           DeliveryAddress
	Status            OrderStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	EstimatedDelivery time.Time
	StatusChangedAt   time.Time
	IsDeleted         bool
}
