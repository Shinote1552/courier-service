package healthcheck_head

import (
	"net/http"
	"sync/atomic"
)

type Handler struct {
	isShuttingDown *atomic.Bool
}

func New(isShuttingDown *atomic.Bool) *Handler {
	return &Handler{
		isShuttingDown: isShuttingDown,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.isShuttingDown.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
