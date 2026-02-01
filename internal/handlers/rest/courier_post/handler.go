package courier_post

import (
	"encoding/json"
	"errors"
	"net/http"

	"service/internal/entities"
	"service/internal/generated/dto"
	"service/internal/service/courier"
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
	var courierModifyDTO dto.CourierCreate
	err := json.NewDecoder(r.Body).Decode(&courierModifyDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	transportType := entities.CourierTransportType(courierModifyDTO.TransportType)
	statusType := entities.CourierStatusType(courierModifyDTO.Status)
	courierModifyEntity := entities.CourierModify{
		Name:          &courierModifyDTO.Name,
		Phone:         &courierModifyDTO.Phone,
		Status:        &statusType,
		TransportType: &transportType,
	}

	id, err := h.service.CreateCourier(r.Context(), courierModifyEntity)
	if err != nil {
		switch {
		case errors.Is(err, courier.ErrMissingRequiredFields),
			errors.Is(err, courier.ErrInvalidName),
			errors.Is(err, courier.ErrInvalidPhone),
			errors.Is(err, courier.ErrInvalidStatus),
			errors.Is(err, courier.ErrInvalidTransport):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, courier.ErrConflict):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	response := dto.CourierCreateResponse{
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("encode JSON response")
	}
}
