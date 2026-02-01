package couriers_get

import (
	"encoding/json"
	"net/http"

	"service/internal/generated/dto"
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
	courierEntities, err := h.service.GetCouriers(r.Context())
	if err != nil {
		// switch {
		// default:
		w.WriteHeader(http.StatusInternalServerError)
		// }
		return
	}

	courierDTOs := make([]dto.Courier, len(courierEntities))
	for i, courier := range courierEntities {
		courierDTOs[i].ID = courier.ID
		courierDTOs[i].Name = courier.Name
		courierDTOs[i].Phone = courier.Phone
		courierDTOs[i].Status = courier.Status.String()
		courierDTOs[i].TransportType = courier.TransportType.String()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(courierDTOs)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("encode JSON response")
	}
}
