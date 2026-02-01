package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nikolaev/service-order/internal/domain/entity"
	"github.com/nikolaev/service-order/internal/handlers/types/convert"
	"github.com/nikolaev/service-order/internal/handlers/types/transport"
	seed "github.com/nikolaev/service-order/internal/usecase/debug/seed"
	uc "github.com/nikolaev/service-order/internal/usecase/order"
)

type OrderHandler struct {
	uc  uc.Service
	dbg seed.Service
}

// NewOrderHandler constructs OrderHandler. Debug seeder is optional.
func NewOrderHandler(uc uc.Service, dbg ...seed.Service) *OrderHandler {
	var d seed.Service
	if len(dbg) > 0 {
		d = dbg[0]
	}
	return &OrderHandler{uc: uc, dbg: d}
}

func (h *OrderHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/order", h.create)
	r.Get("/order/{id}", h.get)
	r.Get("/order/{id}/status", h.getStatus)
	r.Get("/orders", h.list)
	r.Put("/order/{id}", h.update)
	r.Delete("/order/{id}", h.delete)
	// debug route to seed orders
	r.Post("/debug/seed", h.seedDebug)
	return r
}

const (
	HeaderBypass = "X-Bypass-Auth"
	HeaderUserID = "X-User-ID"
)

func (h *OrderHandler) userIDFrom(r *http.Request) string {
	if r.Header.Get(HeaderBypass) == "true" {
		return "default-user"
	}

	return r.Header.Get(HeaderUserID)
}

func (h *OrderHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *OrderHandler) writeError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	s := "internal"
	switch {
	case errors.Is(err, entity.ErrUnauthorized):
		code = http.StatusUnauthorized
		s = "unauthorized"
	case errors.Is(err, entity.ErrForbidden):
		code = http.StatusForbidden
		s = "forbidden"
	case errors.Is(err, entity.ErrInvalidInput), errors.Is(err, entity.ErrInvalidID), errors.Is(err, entity.ErrForeignOwnership):
		code = http.StatusBadRequest
		s = "bad_request"
	case errors.Is(err, entity.ErrNotFound):
		code = http.StatusNotFound
		s = "not_found"
	}
	h.writeJSON(w, code, transport.Error{Code: s, Message: err.Error()})
}

func (h *OrderHandler) create(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	var req transport.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, entity.ErrInvalidInput)
		return
	}
	order, err := h.uc.Create(r.Context(), userID, convert.ToDomainCreate(req))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := convert.ToTransport(order)
	h.writeJSON(w, http.StatusCreated, resp)
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	id := chi.URLParam(r, "id")
	order, err := h.uc.Get(r.Context(), userID, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := convert.ToTransport(order)
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) getStatus(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	id := chi.URLParam(r, "id")
	status, err := h.uc.GetStatus(r.Context(), userID, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"order_id": id, "status": string(status)})
}

func (h *OrderHandler) list(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	if fromStr == "" {
		fromStr = "1970-01-01T00:00:00Z"
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		h.writeError(w, entity.ErrInvalidInput)
		return
	}
	orders, err := h.uc.ListFrom(r.Context(), from)
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := make([]transport.OrderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, convert.ToTransport(o))
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) update(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	id := chi.URLParam(r, "id")
	var req transport.UpdateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, entity.ErrInvalidInput)
		return
	}

	order, err := h.uc.Update(r.Context(), userID, id, convert.ToDomainUpdate(req))
	if err != nil {
		h.writeError(w, err)
		return
	}

	resp := convert.ToTransport(order)
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	id := chi.URLParam(r, "id")
	if err := h.uc.Delete(r.Context(), userID, id); err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, transport.DeleteOrderResponse{
		ID:     id,
		Status: string(entity.OrderStatusDeleted),
	})
}

// seedDebug creates N=10 demo orders for the current user using current time.
func (h *OrderHandler) seedDebug(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFrom(r)
	if h.dbg == nil {
		h.writeError(w, errors.New("debug seeder is not configured"))
		return
	}

	orders, err := h.dbg.Seed(r.Context(), userID, 10)
	if err != nil {
		h.writeError(w, err)
		return
	}

	resp := make([]transport.OrderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, convert.ToTransport(o))
	}
	h.writeJSON(w, http.StatusCreated, resp)
}
