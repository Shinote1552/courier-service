package order_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nikolaev/service-order/internal/domain/entity"
	"github.com/nikolaev/service-order/internal/gateway/kafka"
	"github.com/nikolaev/service-order/internal/handlers"
	"github.com/nikolaev/service-order/internal/handlers/types/transport"
	repo "github.com/nikolaev/service-order/internal/repository/order"
	uc "github.com/nikolaev/service-order/internal/usecase/order"
)

func TestUsecase_Create_Get_Update_Delete(t *testing.T) {
	repository := repo.NewInMemory()
	producer := kafka.NoopProducer{}
	service := uc.New(repository, producer)

	// Create
	order, err := service.Create(context.Background(), "u1", uc.CreateInput{
		OrderNumber:  "N-1",
		FIO:          "Ivanov I.I.",
		RestaurantID: "rest1",
		Items: []entity.Item{
			{
				FoodID:   "f1",
				Name:     "Pizza",
				Quantity: 1,
				Price:    500,
			},
		},
		TotalPrice: 500,
		Address: entity.DeliveryAddress{
			Street: "Main",
		},
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if order.UserID != "u1" || order.RestaurantID != "rest1" {
		t.Fatal("bad order fields")
	}

	// Get
	got, err := service.Get(context.Background(), "u1", order.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got.ID != order.ID {
		t.Fatal("get returned different order")
	}

	// Update
	newFIO := "Petrov P.P."
	upd, err := service.Update(context.Background(), "u1", order.ID, uc.UpdateInput{FIO: &newFIO})
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	if upd.FIO != newFIO {
		t.Fatal("fio not updated")
	}

	// Delete
	if err := service.Delete(context.Background(), "u1", order.ID); err != nil {
		t.Fatalf("delete error: %v", err)
	}
}

func TestHandlers_Create_BadInput(t *testing.T) {
	repository := repo.NewInMemory()
	producer := kafka.NoopProducer{}
	service := uc.New(repository, producer)
	h := handlers.NewOrderHandler(service)
	r := chi.NewRouter()
	r.Mount("/public/api/v1", h.Routes())

	req := httptest.NewRequest("POST", "/public/api/v1/order", strings.NewReader(`{"items":[],"total_price":-1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bypass-Auth", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var er transport.Error
	_ = json.NewDecoder(w.Body).Decode(&er)
	if er.Code == "" {
		t.Fatal("expected error response")
	}
}
