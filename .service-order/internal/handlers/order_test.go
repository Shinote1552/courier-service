package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/nikolaev/service-order/internal/domain/entity"
	"github.com/nikolaev/service-order/internal/handlers"
	"github.com/nikolaev/service-order/internal/handlers/types/transport"
	uc "github.com/nikolaev/service-order/internal/usecase/order"
)

type fakeService struct {
	CreateFn    func(ctx context.Context, userID string, in uc.CreateInput) (*entity.Order, error)
	GetFn       func(ctx context.Context, userID, id string) (*entity.Order, error)
	GetStatusFn func(ctx context.Context, userID, id string) (entity.OrderStatus, error)
	ListFromFn  func(ctx context.Context, from time.Time) ([]*entity.Order, error)
	UpdateFn    func(ctx context.Context, userID, id string, in uc.UpdateInput) (*entity.Order, error)
	DeleteFn    func(ctx context.Context, userID, id string) error
}

func (f fakeService) Create(ctx context.Context, userID string, in uc.CreateInput) (*entity.Order, error) {
	return f.CreateFn(ctx, userID, in)
}

func (f fakeService) Get(ctx context.Context, userID, id string) (*entity.Order, error) {
	return f.GetFn(ctx, userID, id)
}

func (f fakeService) GetStatus(ctx context.Context, userID, id string) (entity.OrderStatus, error) {
	return f.GetStatusFn(ctx, userID, id)
}

func (f fakeService) ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error) {
	return f.ListFromFn(ctx, from)
}

func (f fakeService) Update(ctx context.Context, userID, id string, in uc.UpdateInput) (*entity.Order, error) {
	return f.UpdateFn(ctx, userID, id, in)
}

func (f fakeService) Delete(ctx context.Context, userID, id string) error {
	return f.DeleteFn(ctx, userID, id)
}

func setupRouter(h *handlers.OrderHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/public/api/v1", h.Routes())
	return r
}

func TestOrderHandler_Create_Success(t *testing.T) {
	fake := fakeService{
		CreateFn: func(ctx context.Context, userID string, in uc.CreateInput) (*entity.Order, error) {
			return &entity.Order{
				ID:           "o1",
				UserID:       userID,
				RestaurantID: in.RestaurantID,
				Items:        in.Items,
				TotalPrice:   in.TotalPrice,
				Address:      in.Address,
				Status:       entity.OrderStatusCreated,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}, nil
		},
	}
	h := handlers.NewOrderHandler(fake)
	r := setupRouter(h)

	body := transport.CreateOrderRequest{
		RestaurantID: "rest-1",
		Items: []transport.Item{
			{
				FoodID:   "f1",
				Name:     "Pizza",
				Quantity: 1,
				Price:    500,
			},
		},
		TotalPrice: 500,
		Address: transport.DeliveryAddress{
			Street: "Main",
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/public/api/v1/order", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bypass-Auth", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp transport.OrderResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "o1", resp.ID)
	assert.Equal(t, "rest-1", resp.RestaurantID)
}

func TestOrderHandler_Create_BadJSON(t *testing.T) {
	h := handlers.NewOrderHandler(fakeService{})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/public/api/v1/order", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bypass-Auth", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrderHandler_GetStatus_OK(t *testing.T) {
	fake := fakeService{
		GetStatusFn: func(ctx context.Context, userID, id string) (entity.OrderStatus, error) {
			return entity.OrderStatusPending, nil
		},
	}
	h := handlers.NewOrderHandler(fake)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/public/api/v1/order/abc/status", nil)
	req.Header.Set("X-Bypass-Auth", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var m map[string]string
	_ = json.NewDecoder(w.Body).Decode(&m)
	assert.Equal(t, "abc", m["order_id"])
	assert.Equal(t, "pending", m["status"])
}

func TestOrderHandler_List_OK(t *testing.T) {
	fake := fakeService{
		ListFromFn: func(ctx context.Context, from time.Time) ([]*entity.Order, error) {
			return []*entity.Order{
				{
					ID:           "o1",
					UserID:       "",
					RestaurantID: "r1",
					Items: []entity.Item{
						{
							FoodID: "f1",
							Name:   "P",
						},
					},
					TotalPrice: 100,
					Address:    entity.DeliveryAddress{},
					Status:     entity.OrderStatusCreated,
					CreatedAt:  time.Now().UTC(),
					UpdatedAt:  time.Now().UTC(),
				},
			}, nil
		},
	}
	h := handlers.NewOrderHandler(fake)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/public/api/v1/orders?from=1970-01-01T00:00:00Z", nil)
	req.Header.Set("X-Bypass-Auth", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []transport.OrderResponse
	_ = json.NewDecoder(w.Body).Decode(&list)
	assert.Len(t, list, 1)
	assert.Equal(t, "o1", list[0].ID)
}
