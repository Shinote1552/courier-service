package entities

import "time"

type Order struct {
	ID        string
	Status    OrderStatusType
	CreatedAt time.Time
}

type OrderStatusType string

const (
	OrderCreated   OrderStatusType = "created"
	OrderCancelled OrderStatusType = "cancelled"
	OrderCompleted OrderStatusType = "completed"
)

func (s OrderStatusType) String() string {
	return string(s)
}

type OrderModify struct {
	ID        *string
	Status    *OrderStatusType
	CreatedAt *time.Time
}
