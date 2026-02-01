package courier_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
		service: service,
		log:     handlerLog,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	Id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	courierEntity, err := h.service.GetCourier(r.Context(), Id)
	if err != nil {
		switch {
		case errors.Is(err, courier.ErrCourierNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, courier.ErrInvalidCourierID):
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	courierDTO := dto.Courier{
		ID:            courierEntity.ID,
		Name:          courierEntity.Name,
		Phone:         courierEntity.Phone,
		Status:        courierEntity.Status.String(),
		TransportType: courierEntity.TransportType.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(courierDTO)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("encode JSON response")
	}
}
