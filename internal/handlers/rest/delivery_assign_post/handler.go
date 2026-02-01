package delivery_assign_post

import (
	"encoding/json"
	"errors"
	"net/http"

	"service/internal/generated/dto"
	"service/internal/service/delivery"
	"service/pkg/logger"
)

type Handler struct {
	log     handlerLogger
	service Service
}

func New(log handlerLogger, service Service) *Handler {
	handlerLog := log.With()

	return &Handler{
		log:     handlerLog,
		service: service,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var deliveryAssignDTO dto.DeliveryAssignRequest
	err := json.NewDecoder(r.Body).Decode(&deliveryAssignDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orderID := deliveryAssignDTO.OrderID

	deliveryEntity, err := h.service.DeliveryAssign(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrInvalidOrderID),
			errors.Is(err, delivery.ErrMissingRequiredFields):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, delivery.ErrNoAvailableCouriers),
			errors.Is(err, delivery.ErrOrderAlreadyAssigned):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	response := dto.DeliveryAssignResponse{
		CourierID:        deliveryEntity.CourierID,
		DeliveryDeadline: deliveryEntity.Deadline,
		OrderID:          deliveryEntity.OrderID,
		TransportType:    deliveryEntity.TransportType.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("encode JSON response")
	}
}
