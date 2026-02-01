//go:build integration

package delivery_test

import (
	"context"
	"testing"
	"time"

	"service/internal/entities"
	"service/internal/repository/delivery"
	"service/internal/repository/integration_test"
	service "service/internal/service/delivery"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное создание заказа", func(t *testing.T) {
		actual, err := repo.Create(ctx, entities.DeliveryModify{
			CourierID:  pointer.To(int64(1)),
			OrderID:    pointer.To("some-order-id"),
			CreatedAt:  pointer.To(time.Date(2025, 1, 15, 11, 30, 0, 0, time.UTC)),
			AssignedAt: pointer.To(time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)),
			Deadline:   pointer.To(time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)),
		})
		require.NoError(t, err)
		require.NotNil(t, actual)

		assert.Equal(t, int64(1), actual.CourierID)
		assert.Equal(t, "some-order-id", actual.OrderID)
		assert.WithinDuration(t, time.Date(2025, 1, 15, 11, 30, 0, 0, time.UTC), actual.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), actual.AssignedAt, time.Second)
		assert.WithinDuration(t, time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC), actual.Deadline, time.Second)
	})
}

func TestRepository_Create_OrderAlreadyAssigned(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES (1, 'existing-order', '2025-01-15 11:00:00', '2025-01-15 11:30:00', '2025-01-15 12:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при попытке назначить уже назначенный заказ", func(t *testing.T) {
		actual, err := repo.Create(ctx, entities.DeliveryModify{
			CourierID:  pointer.To(int64(1)),
			OrderID:    pointer.To("existing-order"),
			CreatedAt:  pointer.To(time.Date(2025, 1, 15, 11, 45, 0, 0, time.UTC)),
			AssignedAt: pointer.To(time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)),
			Deadline:   pointer.To(time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)),
		})
		require.Error(t, err)
		require.Nil(t, actual)
		assert.ErrorIs(t, err, service.ErrOrderAlreadyAssigned)
	})
}

func TestRepository_Delete_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES (1, 'order-to-delete', '2025-01-15 11:00:00', '2025-01-15 11:30:00', '2025-01-15 12:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное удаление доставки", func(t *testing.T) {
		err := repo.Delete(ctx, "order-to-delete")
		require.NoError(t, err)

		var count int
		err = q.QueryRow(ctx, "SELECT COUNT(*) FROM delivery WHERE order_id = $1", "order-to-delete").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestRepository_Delete_NotFound(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при удалении несуществующей доставки", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-order")
		require.Error(t, err)
		assert.ErrorIs(t, err, service.ErrDeliveryNotFound)
	})
}

func TestRepository_GetCourierIDAndDeliveryCountByOrderIDForAssing_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', NOW(), NOW());

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES
            (1, 'target-order', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour'),
            (1, 'other-order-1', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour'),
            (1, 'other-order-2', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение ID курьера и количества доставок", func(t *testing.T) {
		courierID, count, err := repo.GetCourierIDAndDeliveryCountByOrderIDForAssing(ctx, "target-order")
		require.NoError(t, err)

		assert.Equal(t, int64(1), courierID)
		assert.Equal(t, int64(2), count)
	})
}

func TestRepository_GetCourierIDAndDeliveryCountByOrderIDForAssing_NotFound(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при поиске несуществующего заказа", func(t *testing.T) {
		courierID, count, err := repo.GetCourierIDAndDeliveryCountByOrderIDForAssing(ctx, "non-existent-order")
		require.Error(t, err)
		assert.ErrorIs(t, err, service.ErrDeliveryNotFound)
		assert.Equal(t, int64(0), courierID)
		assert.Equal(t, int64(0), count)
	})
}

func TestRepository_GetCourierForAssignment_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Courier 1', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
            (2, 'Courier 2', '+79991112234', 'available', 'scooter', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
            (3, 'Courier 3', '+79991112235', 'busy', 'car', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES
            (1, 'order-1', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour'),
            (1, 'order-2', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour'),
            (2, 'order-3', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешный выбор курьера с минимальной нагрузкой", func(t *testing.T) {
		courier, err := repo.GetCourierForAssignment(ctx)
		require.NoError(t, err)
		require.NotNil(t, courier)

		assert.Equal(t, int64(2), courier.ID)
		assert.Equal(t, "Courier 2", courier.Name)
		assert.Equal(t, "+79991112234", courier.Phone)
		assert.Equal(t, entities.CourierAvailable, courier.Status)
		assert.Equal(t, entities.Scooter, courier.TransportType)
	})
}

func TestRepository_GetCourierForAssignment_NoAvailableCouriers(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Courier 1', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
            (2, 'Courier 2', '+79991112234', 'paused', 'scooter', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при отсутствии доступных курьеров", func(t *testing.T) {
		courier, err := repo.GetCourierForAssignment(ctx)
		require.Error(t, err)
		require.Nil(t, courier)
		assert.ErrorIs(t, err, service.ErrNoAvailableCouriers)
	})
}

func TestRepository_UpdateCouriersAvailableWhereDeadlineExpired_Success(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES
		(1, 'Courier 1', '+79991112233', 'busy', 'on_foot', NOW(), NOW()),
		(2, 'Courier 2', '+79991112234', 'busy', 'scooter', NOW(), NOW()),
		(3, 'Courier 3', '+79991112235', 'available', 'car', NOW(), NOW()),
		(4, 'Courier 4', '+79991112236', 'busy', 'car', NOW(), NOW());

		INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
		VALUES
		(1, 'expired-1', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour'),
		(2, 'expired-2', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour'),
		(2, 'active-1', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '30 minutes', NOW() + INTERVAL '30 minutes'),
		(4, 'expired-3', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour');
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное обновление статуса курьеров с истекшими дедлайнами", func(t *testing.T) {
		rowsAffected, err := repo.UpdateCouriersAvailableWhereDeadlineExpired(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), rowsAffected)

		var status1, status2, status3, status4 string

		err = q.QueryRow(ctx, "SELECT status FROM couriers WHERE id = 1").Scan(&status1)
		require.NoError(t, err)
		assert.Equal(t, "available", status1)

		err = q.QueryRow(ctx, "SELECT status FROM couriers WHERE id = 2").Scan(&status2)
		require.NoError(t, err)
		assert.Equal(t, "available", status2)

		err = q.QueryRow(ctx, "SELECT status FROM couriers WHERE id = 3").Scan(&status3)
		require.NoError(t, err)
		assert.Equal(t, "available", status3)

		err = q.QueryRow(ctx, "SELECT status FROM couriers WHERE id = 4").Scan(&status4)
		require.NoError(t, err)
		assert.Equal(t, "available", status4)
	})
}

func TestRepository_GetLastAssignedDeliveryTime_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES
            (1, 'order-1', '2025-01-15 10:00:00', '2025-01-15 10:30:00', '2025-01-15 11:00:00'),
            (1, 'order-2', '2025-01-15 11:00:00', '2025-01-15 11:30:00', '2025-01-15 12:00:00'),
            (1, 'order-3', '2025-01-15 12:00:00', '2025-01-15 12:30:00', '2025-01-15 13:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение времени создания последней доставки из нескольких записей", func(t *testing.T) {
		actual, err := repo.GetLastAssignedDeliveryTime(ctx)
		require.NoError(t, err)

		expected := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
		assert.WithinDuration(t, expected, actual, time.Second)
	})
}

func TestRepository_GetLastAssignedDeliveryTime_SingleDelivery(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES (1, 'single-order', '2025-01-15 14:00:00', '2025-01-15 14:30:00', '2025-01-15 15:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение времени создания единственной доставки в системе", func(t *testing.T) {
		actual, err := repo.GetLastAssignedDeliveryTime(ctx)
		require.NoError(t, err)

		expected := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
		assert.WithinDuration(t, expected, actual, time.Second)
	})
}

func TestRepository_GetLastAssignedDeliveryTime_NotFound(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при отсутствии доставок в базе данных для получения курсора", func(t *testing.T) {
		actual, err := repo.GetLastAssignedDeliveryTime(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, service.ErrDeliveryNotFound)
		assert.True(t, actual.IsZero())
	})
}

func TestRepository_GetLastAssignedDeliveryTime_MultipleDeliveriesDifferentTimes(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier 1', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
            (2, 'Test Courier 2', '+79991112234', 'busy', 'scooter', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES
            (1, 'order-old', '2025-01-15 08:00:00', '2025-01-15 08:30:00', '2025-01-15 09:00:00'),
            (2, 'order-middle', '2025-01-15 10:00:00', '2025-01-15 10:30:00', '2025-01-15 11:00:00'),
            (1, 'order-latest', '2025-01-15 15:30:00', '2025-01-15 16:00:00', '2025-01-15 17:00:00'),
            (2, 'order-recent', '2025-01-15 14:00:00', '2025-01-15 14:30:00', '2025-01-15 15:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение максимального времени создания среди множества доставок", func(t *testing.T) {
		actual, err := repo.GetLastAssignedDeliveryTime(ctx)
		require.NoError(t, err)

		expected := time.Date(2025, 1, 15, 15, 30, 0, 0, time.UTC)
		assert.WithinDuration(t, expected, actual, time.Second)
	})
}

func TestRepository_GetLastAssignedDeliveryTime_WithMicroseconds(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES
            (1, 'order-1', '2025-01-15 12:00:00.123456', '2025-01-15 12:30:00', '2025-01-15 13:00:00'),
            (1, 'order-2', '2025-01-15 12:00:00.987654', '2025-01-15 12:30:00', '2025-01-15 13:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение времени с точностью до микросекунд при наличии близких временных меток", func(t *testing.T) {
		actual, err := repo.GetLastAssignedDeliveryTime(ctx)
		require.NoError(t, err)

		expected := time.Date(2025, 1, 15, 12, 0, 0, 987654000, time.UTC)
		assert.WithinDuration(t, expected, actual, time.Microsecond)
	})
}

func TestRepository_GetCourierIDByOrderID_Success(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'busy', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');

        INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
        VALUES (1, 'test-order-123', '2025-01-15 11:00:00', '2025-01-15 11:30:00', '2025-01-15 12:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Успешное получение ID курьера по order_id", func(t *testing.T) {
		courierID, err := repo.GetCourierIDByOrderID(ctx, "test-order-123")
		require.NoError(t, err)
		assert.Equal(t, int64(1), courierID)
	})
}

func TestRepository_GetCourierIDByOrderID_NotFound(t *testing.T) {
	setupSql := `
        INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
        VALUES
            (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
    `

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := delivery.New(q)
	ctx := context.Background()

	t.Run("Ошибка при поиске несуществующего заказа", func(t *testing.T) {
		courierID, err := repo.GetCourierIDByOrderID(ctx, "non-existent-order")
		require.Error(t, err)
		assert.ErrorIs(t, err, service.ErrDeliveryNotFound)
		assert.Equal(t, int64(0), courierID)
	})
}
