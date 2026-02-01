package token_bucket

import (
	"sync"
	"time"
)

/*
по сути алгоритм простой, реализовываем Allow метод который возвращает true/false,
то есть - мы либо принимаем запрос, либо отклоняем.
*/

// Пример интерфейса
type Limiter interface {
	Allow() bool
}

type TokenBucket struct {
	capacity   int
	tokens     int
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (t *TokenBucket) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.refill()

	if t.tokens > 0 {
		t.tokens--
		return true
	}
	return false
}

func (t *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(t.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	tokensToAdd := int(elapsed * t.refillRate)

	if tokensToAdd > 0 {
		t.tokens += tokensToAdd
		if t.tokens > t.capacity {
			t.tokens = t.capacity
		}
		t.lastRefill = now
	}
}
