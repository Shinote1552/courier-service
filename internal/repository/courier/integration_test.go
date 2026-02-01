//go:build integration

package courier_test

import (
	"context"
	"testing"
	"time"

	"service/internal/entities"
	"service/internal/repository/courier"
	"service/internal/repository/integration_test"
	service "service/internal/service/courier"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create_Success(t *testing.T) {
	integration_test.SetupDB(t, "")
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное создание курьера", func(t *testing.T) {
		status := entities.CourierAvailable
		transport := entities.OnFoot

		id, err := repo.Create(ctx, entities.CourierModify{
			Name:          pointer.To("Test Courier"),
			Phone:         pointer.To("+79991112233"),
			Status:        pointer.To(status),
			TransportType: pointer.To(transport),
		})
		require.NoError(t, err)
		require.Greater(t, id, int64(0))

		var count int
		err = q.QueryRow(ctx, "SELECT COUNT(*) FROM couriers WHERE id = $1", id).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		var name, phone, statusDB, transportTypeDB string
		err = q.QueryRow(ctx, "SELECT name, phone, status, transport_type FROM couriers WHERE id = $1", id).
			Scan(&name, &phone, &statusDB, &transportTypeDB)
		require.NoError(t, err)
		assert.Equal(t, "Test Courier", name)
		assert.Equal(t, "+79991112233", phone)
		assert.Equal(t, "available", statusDB)
		assert.Equal(t, "on_foot", transportTypeDB)
	})
}

func TestRepository_Create_Conflict(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (name, phone, status, transport_type, created_at, updated_at)
		VALUES ('Existing Courier', '+79991112233', 'available', 'on_foot', NOW(), NOW());
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Ошибка при создании курьера с существующим телефоном", func(t *testing.T) {
		status := entities.CourierAvailable
		transport := entities.OnFoot

		id, err := repo.Create(ctx, entities.CourierModify{
			Name:          pointer.To("Another Courier"),
			Phone:         pointer.To("+79991112233"),
			Status:        pointer.To(status),
			TransportType: pointer.To(transport),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, service.ErrConflict)
		assert.Equal(t, int64(0), id)
	})
}

func TestRepository_Update_Success(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES (1, 'Old Name', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное обновление курьера", func(t *testing.T) {
		newStatus := entities.CourierBusy
		newTransport := entities.Scooter
		newName := "Updated Name"
		newPhone := "+79991112234"

		updatedCourier, err := repo.Update(ctx, entities.CourierModify{
			ID:            pointer.To(int64(1)),
			Name:          &newName,
			Phone:         &newPhone,
			Status:        &newStatus,
			TransportType: &newTransport,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedCourier)

		assert.Equal(t, int64(1), updatedCourier.ID)
		assert.Equal(t, "Updated Name", updatedCourier.Name)
		assert.Equal(t, "+79991112234", updatedCourier.Phone)
		assert.Equal(t, entities.CourierBusy, updatedCourier.Status)
		assert.Equal(t, entities.Scooter, updatedCourier.TransportType)
		assert.NotEqual(t, updatedCourier.CreatedAt, updatedCourier.UpdatedAt)

		var name, phone, statusDB, transportTypeDB string
		var updatedAt time.Time
		err = q.QueryRow(ctx, "SELECT name, phone, status, transport_type, updated_at FROM couriers WHERE id = 1").
			Scan(&name, &phone, &statusDB, &transportTypeDB, &updatedAt)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", name)
		assert.Equal(t, "+79991112234", phone)
		assert.Equal(t, "busy", statusDB)
		assert.Equal(t, "scooter", transportTypeDB)
		assert.True(t, updatedAt.After(time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)))
	})
}

func TestRepository_Update_Partial(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное частичное обновление курьера (только имя)", func(t *testing.T) {
		newName := "New Name Only"

		updatedCourier, err := repo.Update(ctx, entities.CourierModify{
			ID:   pointer.To(int64(1)),
			Name: &newName,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedCourier)

		assert.Equal(t, int64(1), updatedCourier.ID)
		assert.Equal(t, "New Name Only", updatedCourier.Name)
		assert.Equal(t, "+79991112233", updatedCourier.Phone)
		assert.Equal(t, entities.CourierAvailable, updatedCourier.Status)
		assert.Equal(t, entities.OnFoot, updatedCourier.TransportType)
	})
}

func TestRepository_Update_NotFound(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES (1, 'Test Courier', '+79991112233', 'available', 'on_foot', NOW(), NOW());
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Ошибка при обновлении несуществующего курьера", func(t *testing.T) {
		newName := "Updated Name"
		nonExistentID := int64(999)

		updatedCourier, err := repo.Update(ctx, entities.CourierModify{
			ID:   &nonExistentID,
			Name: &newName,
		})
		require.Error(t, err)
		require.Nil(t, updatedCourier)
		assert.ErrorIs(t, err, service.ErrCourierNotFound)
	})
}

func TestRepository_Update_Conflict(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES 
			(1, 'Courier 1', '+79991112233', 'available', 'on_foot', NOW(), NOW()),
			(2, 'Courier 2', '+79991112234', 'available', 'scooter', NOW(), NOW());
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Ошибка при обновлении телефона на уже существующий", func(t *testing.T) {
		existingPhone := "+79991112234"

		updatedCourier, err := repo.Update(ctx, entities.CourierModify{
			ID:    pointer.To(int64(1)),
			Phone: &existingPhone,
		})
		require.Error(t, err)
		require.Nil(t, updatedCourier)
		assert.ErrorIs(t, err, service.ErrConflict)
	})
}

func TestRepository_GetByID_Success(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES (1, 'Test Courier', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное получение курьера по ID", func(t *testing.T) {
		courier, err := repo.GetByID(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, courier)

		assert.Equal(t, int64(1), courier.ID)
		assert.Equal(t, "Test Courier", courier.Name)
		assert.Equal(t, "+79991112233", courier.Phone)
		assert.Equal(t, entities.CourierAvailable, courier.Status)
		assert.Equal(t, entities.OnFoot, courier.TransportType)
		assert.Equal(t, time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC), courier.CreatedAt)
		assert.Equal(t, time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC), courier.UpdatedAt)
	})
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	integration_test.SetupDB(t, "")
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Ошибка при получении несуществующего курьера", func(t *testing.T) {
		courier, err := repo.GetByID(ctx, 999)
		require.Error(t, err)
		require.Nil(t, courier)
		assert.ErrorIs(t, err, service.ErrCourierNotFound)
	})
}

func TestRepository_GetAll_Success(t *testing.T) {
	setupSql := `
		INSERT INTO couriers (id, name, phone, status, transport_type, created_at, updated_at)
		VALUES 
			(1, 'Courier 1', '+79991112233', 'available', 'on_foot', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
			(2, 'Courier 2', '+79991112234', 'busy', 'scooter', '2025-01-15 11:00:00', '2025-01-15 11:00:00'),
			(3, 'Courier 3', '+79991112235', 'paused', 'car', '2025-01-15 11:00:00', '2025-01-15 11:00:00');
	`

	integration_test.SetupDB(t, setupSql)
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное получение всех курьеров", func(t *testing.T) {
		couriers, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Len(t, couriers, 3)

		assert.Equal(t, int64(1), couriers[0].ID)
		assert.Equal(t, "Courier 1", couriers[0].Name)
		assert.Equal(t, entities.CourierAvailable, couriers[0].Status)

		assert.Equal(t, int64(2), couriers[1].ID)
		assert.Equal(t, "Courier 2", couriers[1].Name)
		assert.Equal(t, entities.CourierBusy, couriers[1].Status)

		assert.Equal(t, int64(3), couriers[2].ID)
		assert.Equal(t, "Courier 3", couriers[2].Name)
		assert.Equal(t, entities.CourierPaused, couriers[2].Status)
	})
}

func TestRepository_GetAll_Empty(t *testing.T) {
	integration_test.SetupDB(t, "")
	defer integration_test.TeardownDB(t)

	q := integration_test.GetQuerier()
	repo := courier.New(q)
	ctx := context.Background()

	t.Run("Успешное получение пустого списка курьеров", func(t *testing.T) {
		couriers, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Empty(t, couriers)
		assert.Len(t, couriers, 0)
	})
}
