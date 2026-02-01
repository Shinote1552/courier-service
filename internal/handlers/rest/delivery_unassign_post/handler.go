package delivery_unassign_post

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
	var deliveryUnassignDTO dto.DeliveryUnassignRequest
	err := json.NewDecoder(r.Body).Decode(&deliveryUnassignDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orderID := deliveryUnassignDTO.OrderID

	deliveryEntity, err := h.service.DeliveryUnassign(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrInvalidOrderID),
			errors.Is(err, delivery.ErrMissingRequiredFields),
			errors.Is(err, delivery.ErrCourierHasActiveDeliveries):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, delivery.ErrDeliveryNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	response := dto.DeliveryUnassignResponse{
		CourierID: deliveryEntity.CourierID,
		OrderID:   deliveryEntity.OrderID,
		Status:    deliveryEntity.Status,
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
