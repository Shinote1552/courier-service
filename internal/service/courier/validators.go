package courier

import "strings"

func isValidName(name string) bool {
	return strings.TrimSpace(name) != ""
}

func isValidPhone(phone string) bool {
	phone = strings.TrimSpace((phone))
	if !strings.HasPrefix(phone, "+") {
		return false
	}

	for _, char := range phone[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func isValidStatus(status string) bool {
	switch status {
	case "available", "busy", "paused":
		return true
	default:
		return false
	}
}

func isValidTransport(status string) bool {
	switch status {
	case "on_foot", "scooter", "car":
		return true
	default:
		return false
	}
}
