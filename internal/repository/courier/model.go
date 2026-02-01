package courier

import (
	"time"
)

type CourierDB struct {
	ID            int64
	Name          string
	Phone         string
	Status        string
	TransportType string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CourierModifyDB struct {
	ID            *int64
	Name          *string
	Phone         *string
	Status        *string
	TransportType *string
}
