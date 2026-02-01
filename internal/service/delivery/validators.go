package delivery

import "strings"

func isValidOrderID(orderID string) bool {
	return strings.TrimSpace(orderID) != ""
}
