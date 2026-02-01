package entities

import (
	"time"
)

type Courier struct {
	ID            int64
	Name          string
	Phone         string
	Status        CourierStatusType
	TransportType CourierTransportType
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CourierTransportType string

const (
	OnFoot  CourierTransportType = "on_foot"
	Scooter CourierTransportType = "scooter"
	Car     CourierTransportType = "car"
)

const DefaultTransportType = OnFoot

func (t CourierTransportType) String() string {
	return string(t)
}

type CourierStatusType string

const (
	CourierAvailable CourierStatusType = "available"
	CourierBusy      CourierStatusType = "busy"
	CourierPaused    CourierStatusType = "paused"
)

const DefaultStatusType = CourierAvailable

func (t CourierStatusType) String() string {
	return string(t)
}

type CourierModify struct {
	ID            *int64
	Name          *string
	Phone         *string
	Status        *CourierStatusType
	TransportType *CourierTransportType
}
