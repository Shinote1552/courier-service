package delivery_deadline

import (
	"time"

	"service/internal/entities"
)

type DeliveryTimeFactory struct{}

func New() *DeliveryTimeFactory {
	return &DeliveryTimeFactory{}
}

func (d *DeliveryTimeFactory) CalculateDeadline(transportType entities.CourierTransportType, baseTime time.Time) time.Time {
	resultTime := baseTime
	switch transportType {
	case entities.OnFoot:
		resultTime = resultTime.Add(time.Minute * 15)
	case entities.Scooter:
		resultTime = resultTime.Add(time.Minute * 10)
	case entities.Car:
		resultTime = resultTime.Add(time.Minute * 5)
	default:
		resultTime = resultTime.Add(time.Minute * 15)
	}

	return resultTime
}
