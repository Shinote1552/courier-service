package courier_put

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
	var courierModifyDTO dto.CourierUpdate
	err := json.NewDecoder(r.Body).Decode(&courierModifyDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	courierModifyEntity := entities.CourierModify{
		ID: &courierModifyDTO.ID,
	}

	// Опциональные параметры
	if courierModifyDTO.Name != nil {
		courierModifyEntity.Name = courierModifyDTO.Name
	}
	if courierModifyDTO.Phone != nil {
		courierModifyEntity.Phone = courierModifyDTO.Phone
	}
	if courierModifyDTO.Status != nil {
		statusType := entities.CourierStatusType(*courierModifyDTO.Status)
		courierModifyEntity.Status = &statusType
	}
	if courierModifyDTO.TransportType != nil {
		transportType := entities.CourierTransportType(*courierModifyDTO.TransportType)
		courierModifyEntity.TransportType = &transportType
	}

	res, err := h.service.UpdateCourier(r.Context(), courierModifyEntity)
	if err != nil {
		switch {
		case errors.Is(err, courier.ErrMissingRequiredFields),
			errors.Is(err, courier.ErrInvalidCourierID),
			errors.Is(err, courier.ErrInvalidName),
			errors.Is(err, courier.ErrInvalidPhone),
			errors.Is(err, courier.ErrInvalidStatus),
			errors.Is(err, courier.ErrInvalidTransport):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, courier.ErrCourierNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, courier.ErrConflict):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	response := dto.Courier{
		ID:            res.ID,
		Name:          res.Name,
		Phone:         res.Phone,
		Status:        res.Status.String(),
		TransportType: res.TransportType.String(),
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
