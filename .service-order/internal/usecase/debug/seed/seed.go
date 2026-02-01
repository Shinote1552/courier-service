package seed

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

func (s *service) Seed(ctx context.Context, userID string, n int) ([]*entity.Order, error) {
	if n <= 0 {
		n = 10
	}
	if userID == "" {
		userID = "default-user"
	}

	out := make([]*entity.Order, 0, n)
	statuses := []entity.OrderStatus{
		entity.OrderStatusCreated,
		entity.OrderStatusPending,
		entity.OrderStatusConfirmed,
		entity.OrderStatusCooking,
		entity.OrderStatusDelivering,
		entity.OrderStatusDelivered,
		entity.OrderStatusCanceled,
		entity.OrderStatusUpdated,
		entity.OrderStatusCompleted,
		entity.OrderStatusPending,
	}

	for i := 0; i < n; i++ {
		st := statuses[i%len(statuses)]
		createdAt := time.Now()
		updatedAt := createdAt
		statusChangedAt := createdAt
		estimated := createdAt.Add(minutes(30 + i))

		items := []entity.Item{
			{
				FoodID:   "f-" + itoa(i, 1),
				Name:     pick([]string{"Burger", "Pizza", "Sushi", "Pasta"}, i),
				Quantity: 1 + (i % 3),
				Price:    300 + 50*(i%5),
			},
			{
				FoodID:   "d-" + itoa(i, 2),
				Name:     pick([]string{"Cola", "Tea", "Juice"}, i+1),
				Quantity: 1,
				Price:    120 + 10*(i%4),
			},
		}
		total := int64(0)
		for _, it := range items {
			total += int64(it.Price * it.Quantity)
		}
		addr := entity.DeliveryAddress{
			Street:    pick([]string{"Lenina", "Tverskaya", "Nevsky", "Arbat"}, i),
			House:     itoa(10+i, 0),
			Apartment: itoa(20+i*3, 0),
			Floor:     itoa(1+(i%10), 0),
			Comment:   pick([]string{"Позвонить за 5 минут", "Код домофона 1234", "Оставить у двери"}, i),
		}

		switch st {
		case entity.OrderStatusPending:
			statusChangedAt = createdAt.Add(seconds(2))
			updatedAt = statusChangedAt
		case entity.OrderStatusConfirmed:
			statusChangedAt = createdAt.Add(seconds(7))
			updatedAt = statusChangedAt
		case entity.OrderStatusCooking:
			statusChangedAt = createdAt.Add(seconds(12))
			updatedAt = statusChangedAt
		case entity.OrderStatusDelivering:
			statusChangedAt = createdAt.Add(minutes(6))
			updatedAt = statusChangedAt
		case entity.OrderStatusDelivered, entity.OrderStatusCompleted:
			statusChangedAt = createdAt.Add(minutes(20))
			updatedAt = statusChangedAt
		case entity.OrderStatusCanceled, entity.OrderStatusDeleted:
			statusChangedAt = createdAt.Add(seconds(3))
			updatedAt = statusChangedAt
		case entity.OrderStatusUpdated:
			statusChangedAt = createdAt.Add(seconds(15))
			updatedAt = statusChangedAt
		}

		o := &entity.Order{
			ID:                uuid.NewString(),
			UserID:            userID,
			OrderNumber:       sprintf("%06d", 1000+i),
			FIO:               pick([]string{"Иван Иванов", "Петр Петров", "Анна Смирнова", "John Doe"}, i),
			RestaurantID:      pick([]string{"rest-1", "rest-2", "rest-3"}, i),
			Items:             items,
			TotalPrice:        total,
			Address:           addr,
			Status:            st,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
			EstimatedDelivery: estimated,
			StatusChangedAt:   statusChangedAt,
			IsDeleted:         st == entity.OrderStatusDeleted,
		}
		if err := s.repo.Create(ctx, o); err != nil {
			return nil, err
		}
		out = append(out, o)
	}

	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out, nil
}

func itoa(v int, salt int) string    { return fmtInt(v + salt) }
func sprintf(_ string, v int) string { return fmtPadded(v) }

func pick[T any](arr []T, i int) T { return arr[i%len(arr)] }

func seconds(s int) time.Duration { return time.Duration(s) * time.Second }
func minutes(m int) time.Duration { return time.Duration(m) * time.Minute }

func fmtInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		d := byte('0' + (v % 10))
		buf = append([]byte{d}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func fmtPadded(v int) string {
	s := fmtInt(v)
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}
