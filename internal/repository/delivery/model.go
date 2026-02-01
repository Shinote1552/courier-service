package delivery

import "time"

type DeliveryDB struct {
	ID         int64
	CourierID  int64
	OrderID    string
	CreatedAt  time.Time
	AssignedAt time.Time
	Deadline   time.Time
}

type DeliveryModifyDB struct {
	ID         *int64
	CourierID  *int64
	OrderID    *string
	CreatedAt  *time.Time
	AssignedAt *time.Time
	Deadline   *time.Time
}

type AvailableCourierDB struct {
	ID            int64
	Name          string
	Phone         string
	Status        string
	TransportType string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
