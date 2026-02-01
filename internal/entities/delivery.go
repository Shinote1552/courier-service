package entities

import "time"

type Delivery struct {
	ID         int64
	CourierID  int64
	OrderID    string
	CreatedAt  time.Time
	AssignedAt time.Time
	Deadline   time.Time
}

type DeliveryModify struct {
	ID         *int64
	CourierID  *int64
	OrderID    *string
	CreatedAt  *time.Time
	AssignedAt *time.Time
	Deadline   *time.Time
}

type DeliveryAssignment struct {
	CourierID     int64
	OrderID       string
	AssignedAt    time.Time
	Deadline      time.Time
	TransportType CourierTransportType
}

type DeliveryUnassignment struct {
	CourierID int64
	OrderID   string
	Status    string
}
