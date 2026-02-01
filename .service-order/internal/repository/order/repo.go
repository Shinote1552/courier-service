package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

type InMemory struct {
	mu    sync.RWMutex
	store map[string]*entity.Order
}

func (r *InMemory) ListFrom(_ context.Context, from time.Time) ([]*entity.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.Order, 0)
	for _, o := range r.store {
		if o.IsDeleted {
			continue
		}
		if o.CreatedAt.Before(from) {
			continue
		}
		copy := *o
		out = append(out, &copy)
	}
	return out, nil
}

func NewInMemory() *InMemory {
	return &InMemory{store: make(map[string]*entity.Order)}
}

func (r *InMemory) Create(_ context.Context, o *entity.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[o.ID]; ok {
		return errors.New("duplicate id")
	}
	copy := *o
	r.store[o.ID] = &copy
	return nil
}

func (r *InMemory) GetByID(_ context.Context, id string) (*entity.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if o, ok := r.store[id]; ok {
		copy := *o
		return &copy, nil
	}
	return nil, entity.ErrNotFound
}

func (r *InMemory) Update(_ context.Context, o *entity.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[o.ID]; !ok {
		return entity.ErrNotFound
	}
	copy := *o
	r.store[o.ID] = &copy
	return nil
}

func (r *InMemory) MarkDeleted(_ context.Context, id string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.store[id]
	if !ok {
		return entity.ErrNotFound
	}
	if o.UserID != userID {
		return entity.ErrForeignOwnership
	}
	o.IsDeleted = true
	o.Status = entity.OrderStatusDeleted
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// AdvanceStatuses updates order statuses based on elapsed time.
// Rules:
//
//	created -> after 1s -> pending
//	pending -> after 5s -> confirmed
//	confirmed -> after 5s -> cooking
//	cooking -> after 5m -> delivering
//	delivering -> after 10m -> completed
func (r *InMemory) AdvanceStatuses(now time.Time) []*entity.Order {
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := make([]*entity.Order, 0)
	for _, o := range r.store {
		if o.IsDeleted || o.Status == entity.OrderStatusCanceled || o.Status == entity.OrderStatusCompleted {
			continue
		}
		last := o.StatusChangedAt
		if last.IsZero() {
			last = o.CreatedAt
		}
		prev := o.Status
		switch o.Status {
		case entity.OrderStatusCreated:
			if now.Sub(last) >= 1*time.Second {
				o.Status = entity.OrderStatusPending
				o.StatusChangedAt = now
				o.UpdatedAt = now
			}
		case entity.OrderStatusPending:
			if now.Sub(last) >= 5*time.Second {
				o.Status = entity.OrderStatusConfirmed
				o.StatusChangedAt = now
				o.UpdatedAt = now
			}
		case entity.OrderStatusConfirmed:
			if now.Sub(last) >= 5*time.Second {
				o.Status = entity.OrderStatusCooking
				o.StatusChangedAt = now
				o.UpdatedAt = now
			}
		case entity.OrderStatusCooking:
			if now.Sub(last) >= 5*time.Minute {
				o.Status = entity.OrderStatusDelivering
				o.StatusChangedAt = now
				o.UpdatedAt = now
			}
		case entity.OrderStatusDelivering:
			if now.Sub(last) >= 10*time.Minute {
				o.Status = entity.OrderStatusCompleted
				o.StatusChangedAt = now
				o.UpdatedAt = now
			}
		}
		if o.Status != prev {
			cp := *o
			changed = append(changed, &cp)
		}
	}
	return changed
}
