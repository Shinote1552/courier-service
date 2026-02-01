package order

import "errors"

var (
	ErrStatusMismatch  = errors.New("order status mismatch between event and order-service")
	ErrUndefinedStatus = errors.New("undefined order status")
	ErrOrderNotFound   = errors.New("order not found")
)
