package token_bucket_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"service/pkg/token_bucket"
)

func TestTokenBucket_Allow_BasicBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		capacity       int
		refillRate     float64
		requestCount   int
		expectedAllows int
	}{
		{
			name:           "Все запросы проходят в пределах capacity",
			capacity:       5,
			refillRate:     10.0,
			requestCount:   5,
			expectedAllows: 5,
		},
		{
			name:           "Превышение capacity блокирует лишние запросы",
			capacity:       3,
			refillRate:     10.0,
			requestCount:   5,
			expectedAllows: 3,
		},
		{
			name:           "Нулевой capacity блокирует все запросы",
			capacity:       0,
			refillRate:     10.0,
			requestCount:   3,
			expectedAllows: 0,
		},
		{
			name:           "Единичная емкость пропускает только первый запрос",
			capacity:       1,
			refillRate:     5.0,
			requestCount:   3,
			expectedAllows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := token_bucket.NewTokenBucket(tt.capacity, tt.refillRate)

			allowed := 0
			for i := 0; i < tt.requestCount; i++ {
				if tb.Allow() {
					allowed++
				}
			}

			assert.Equal(t, tt.expectedAllows, allowed)
		})
	}
}

func TestTokenBucket_Refill_TimeBased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		capacity        int
		refillRate      float64
		initialRequests int
		sleepDuration   time.Duration
		afterSleep      int
		expectedMin     int
		expectedMax     int
	}{
		{
			name:            "Пополнение после полного исчерпания токенов",
			capacity:        10,
			refillRate:      10.0,
			initialRequests: 10,
			sleepDuration:   250 * time.Millisecond,
			afterSleep:      3,
			expectedMin:     2,
			expectedMax:     3,
		},
		{
			name:            "Частичное пополнение при дробном времени",
			capacity:        5,
			refillRate:      20.0,
			initialRequests: 5,
			sleepDuration:   100 * time.Millisecond,
			afterSleep:      3,
			expectedMin:     2,
			expectedMax:     2,
		},
		{
			name:            "Пополнение не превышает capacity",
			capacity:        3,
			refillRate:      100.0,
			initialRequests: 3,
			sleepDuration:   50 * time.Millisecond,
			afterSleep:      5,
			expectedMin:     3,
			expectedMax:     3,
		},
		{
			name:            "Нулевая скорость пополнения блокирует восстановление",
			capacity:        5,
			refillRate:      0.0,
			initialRequests: 5,
			sleepDuration:   50 * time.Millisecond,
			afterSleep:      3,
			expectedMin:     0,
			expectedMax:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := token_bucket.NewTokenBucket(tt.capacity, tt.refillRate)

			for i := 0; i < tt.initialRequests; i++ {
				tb.Allow()
			}

			time.Sleep(tt.sleepDuration)

			allowed := 0
			for i := 0; i < tt.afterSleep; i++ {
				if tb.Allow() {
					allowed++
				}
			}

			assert.GreaterOrEqual(t, allowed, tt.expectedMin,
				"Expected at least %d allowed requests", tt.expectedMin)
			assert.LessOrEqual(t, allowed, tt.expectedMax,
				"Expected at most %d allowed requests", tt.expectedMax)
		})
	}
}

func TestTokenBucket_BurstAndRefill(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		capacity          int
		refillRate        float64
		initialBurst      int
		expectedBurst     int
		sleepDuration     time.Duration
		afterSleepRequest int
		expectedMin       int
		expectedMax       int
	}{
		{
			name:              "Субсекундное пополнение после burst",
			capacity:          10,
			refillRate:        20.0,
			initialBurst:      10,
			expectedBurst:     10,
			sleepDuration:     150 * time.Millisecond,
			afterSleepRequest: 5,
			expectedMin:       3,
			expectedMax:       3,
		},
		{
			name:              "Быстрый burst с частичным пополнением",
			capacity:          10,
			refillRate:        10.0,
			initialBurst:      15,
			expectedBurst:     10,
			sleepDuration:     200 * time.Millisecond,
			afterSleepRequest: 5,
			expectedMin:       2,
			expectedMax:       2,
		},
		{
			name:              "Пополнение до capacity после ожидания",
			capacity:          5,
			refillRate:        50.0,
			initialBurst:      5,
			expectedBurst:     5,
			sleepDuration:     120 * time.Millisecond,
			afterSleepRequest: 10,
			expectedMin:       5,
			expectedMax:       5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := token_bucket.NewTokenBucket(tt.capacity, tt.refillRate)

			allowed := 0
			for i := 0; i < tt.initialBurst; i++ {
				if tb.Allow() {
					allowed++
				}
			}

			assert.Equal(t, tt.expectedBurst, allowed,
				"Burst должен пропустить только capacity запросов")

			time.Sleep(tt.sleepDuration)

			additionalAllowed := 0
			for i := 0; i < tt.afterSleepRequest; i++ {
				if tb.Allow() {
					additionalAllowed++
				}
			}

			assert.GreaterOrEqual(t, additionalAllowed, tt.expectedMin,
				"После пополнения должны пройти минимум %d запросов", tt.expectedMin)
			assert.LessOrEqual(t, additionalAllowed, tt.expectedMax,
				"После пополнения должны пройти максимум %d запросов", tt.expectedMax)
		})
	}
}

func TestTokenBucket_Concurrent_ThreadSafety(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		capacity     int
		refillRate   float64
		goroutines   int
		requestsEach int
	}{
		{
			name:         "Конкурентный доступ 10 горутин по 5 запросов",
			capacity:     20,
			refillRate:   0.0,
			goroutines:   10,
			requestsEach: 5,
		},
		{
			name:         "Высокая конкуренция 50 горутин по 10 запросов",
			capacity:     100,
			refillRate:   0.0,
			goroutines:   50,
			requestsEach: 10,
		},
		{
			name:         "Высокая конкуренция без пополнения",
			capacity:     1000,
			refillRate:   0.0,
			goroutines:   100,
			requestsEach: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := token_bucket.NewTokenBucket(tt.capacity, tt.refillRate)

			var wg sync.WaitGroup
			var allowedCount atomic.Int64
			var deniedCount atomic.Int64

			for i := 0; i < tt.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < tt.requestsEach; j++ {
						if tb.Allow() {
							allowedCount.Add(1)
						} else {
							deniedCount.Add(1)
						}
					}
				}()
			}

			wg.Wait()

			totalRequests := tt.goroutines * tt.requestsEach
			assert.Equal(t, int64(totalRequests), allowedCount.Load()+deniedCount.Load(),
				"Все запросы должны быть учтены")
			assert.LessOrEqual(t, allowedCount.Load(), int64(tt.capacity),
				"Разрешенных не больше capacity")
		})
	}
}

func TestTokenBucket_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		capacity   int
		refillRate float64
		operation  func(tb *token_bucket.TokenBucket) bool
		expected   bool
	}{
		{
			name:       "Дробная скорость пополнения 0.5 токенов/сек",
			capacity:   5,
			refillRate: 2.0,
			operation: func(tb *token_bucket.TokenBucket) bool {
				tb.Allow()
				tb.Allow()
				time.Sleep(600 * time.Millisecond)
				return tb.Allow()
			},
			expected: true,
		},
		{
			name:       "Очень высокая скорость пополнения 1000 токенов/сек",
			capacity:   10,
			refillRate: 1000.0,
			operation: func(tb *token_bucket.TokenBucket) bool {
				for i := 0; i < 10; i++ {
					tb.Allow()
				}
				time.Sleep(50 * time.Millisecond)
				return tb.Allow()
			},
			expected: true,
		},
		{
			name:       "Capacity = 1 с медленным пополнением",
			capacity:   1,
			refillRate: 5.0,
			operation: func(tb *token_bucket.TokenBucket) bool {
				tb.Allow()
				time.Sleep(250 * time.Millisecond)
				return tb.Allow()
			},
			expected: true,
		},
		{
			name:       "Очень медленная скорость пополнения 0.0003 токен/сек",
			capacity:   1,
			refillRate: 0.0003,
			operation: func(tb *token_bucket.TokenBucket) bool {
				require.True(t, tb.Allow())
				require.False(t, tb.Allow())
				time.Sleep(100 * time.Millisecond)
				return tb.Allow()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := token_bucket.NewTokenBucket(tt.capacity, tt.refillRate)
			result := tt.operation(tb)

			assert.Equal(t, tt.expected, result)
		})
	}
}
