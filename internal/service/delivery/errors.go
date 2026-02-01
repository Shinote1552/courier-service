package delivery

import "errors"

var (
	ErrMissingRequiredFields = errors.New("missing required fields")
	ErrInvalidOrderID        = errors.New("invalid order id")
	ErrInvalidCourierID      = errors.New("invalid courier id")

	ErrNoAvailableCouriers        = errors.New("no available couriers")
	ErrDeliveryNotFound           = errors.New("delivery not found")
	ErrOrderAlreadyAssigned       = errors.New("order already assigned")
	ErrCourierHasActiveDeliveries = errors.New("courier has active deliveries")
)
