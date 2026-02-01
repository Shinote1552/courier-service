package ping_get

import (
	"encoding/json"
	"net/http"

	"service/internal/generated/dto"
	"service/pkg/logger"
)

type Handler struct {
	log handlerLogger
}

func New(log handlerLogger) *Handler {
	handlerLog := log.With()

	return &Handler{
		log: handlerLog,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	message := "pong"
	res := dto.PingResponse{
		Message: &message,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("encode JSON response")
	}
}
