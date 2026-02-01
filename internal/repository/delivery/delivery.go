package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"service/internal/entities"
	"service/internal/repository"
	"service/internal/service/delivery"
)

type Repository struct {
	querier Querier
}

func New(querier Querier) *Repository {
	return &Repository{
		querier: querier,
	}
}

func (r *Repository) Create(ctx context.Context, deliveryModify entities.DeliveryModify) (*entities.Delivery, error) {
	deliveryModifyDB := FromDomainModify(&deliveryModify)

	query := `
		INSERT INTO delivery (courier_id, order_id, created_at, assigned_at, deadline)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, courier_id, order_id, created_at, assigned_at, deadline
	`

	var deliveryDB DeliveryDB
	err := r.querier.QueryRow(
		ctx,
		query,
		deliveryModifyDB.CourierID,
		deliveryModifyDB.OrderID,
		deliveryModifyDB.CreatedAt,
		deliveryModifyDB.AssignedAt,
		deliveryModifyDB.Deadline,
	).Scan(
		&deliveryDB.ID,
		&deliveryDB.CourierID,
		&deliveryDB.OrderID,
		&deliveryDB.CreatedAt,
		&deliveryDB.AssignedAt,
		&deliveryDB.Deadline,
	)
	if err != nil {
		if repository.IsPgErrorWithCode(err, repository.PgErrUniqueViolation) {
			return nil, delivery.ErrOrderAlreadyAssigned
		}
		return nil, fmt.Errorf("unexpected delivery repository create error: %w", err)
	}

	deliveryDomain := ToDomain(&deliveryDB)
	return deliveryDomain, nil
}

func (r *Repository) Delete(ctx context.Context, orderID string) error {
	query := `
		DELETE FROM delivery WHERE order_id = $1
	`
	result, err := r.querier.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("unexpected delivery repository delete error: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return delivery.ErrDeliveryNotFound
	}

	return nil
}

func (r *Repository) GetCourierIDAndDeliveryCountByOrderIDForAssing(ctx context.Context, orderID string) (courierID, activeDeliveriesCount int64, err error) {
	query := `
        SELECT 
            d1.courier_id,
            COUNT(d2.id) FILTER (WHERE d2.deadline >= NOW())
        FROM delivery d1
        LEFT JOIN delivery d2 
            ON d2.courier_id = d1.courier_id 
            AND d2.order_id != $1
        WHERE d1.order_id = $1
        GROUP BY d1.courier_id
	`

	err = r.querier.QueryRow(ctx, query, orderID).Scan(&courierID, &activeDeliveriesCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, delivery.ErrDeliveryNotFound
		}
		return 0, 0, fmt.Errorf("unexpected delivery repository get courier id error: %w", err)
	}
	return courierID, activeDeliveriesCount, nil
}

func (r *Repository) GetCourierForAssignment(ctx context.Context) (*entities.Courier, error) {
	query := `
        SELECT 
            c.id, c.name, c.phone, c.status, c.transport_type, c.created_at, c.updated_at
        FROM couriers c
        LEFT JOIN delivery d ON d.courier_id = c.id
        WHERE c.status = 'available'
        GROUP BY c.id
        ORDER BY COUNT(d.id) FILTER (WHERE d.deadline >= NOW()) ASC, c.id ASC
        LIMIT 1
	`

	var courierDB AvailableCourierDB
	err := r.querier.QueryRow(ctx, query).Scan(
		&courierDB.ID,
		&courierDB.Name,
		&courierDB.Phone,
		&courierDB.Status,
		&courierDB.TransportType,
		&courierDB.CreatedAt,
		&courierDB.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, delivery.ErrNoAvailableCouriers
		}
		return nil, fmt.Errorf("unexpected delivery repository find courier error: %w", err)
	}

	courierEntity := ToCourierDomain(&courierDB)
	return courierEntity, nil
}

func (r *Repository) UpdateCouriersAvailableWhereDeadlineExpired(ctx context.Context) (int64, error) {
	query := `
        UPDATE couriers 
        SET status = 'available',
            updated_at = NOW()
        WHERE status = 'busy' 
        AND EXISTS (
            SELECT 1 
            FROM delivery 
            WHERE delivery.courier_id = couriers.id 
              AND delivery.deadline < NOW()
        )
    `

	result, err := r.querier.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("unexpected delivery repository release expired couriers error: %w", err)
	}

	return result.RowsAffected(), nil
}

func (r *Repository) GetLastAssignedDeliveryTime(ctx context.Context) (time.Time, error) {
	query := `
		SELECT MAX(created_at)
		FROM delivery
	`

	var lastTime *time.Time
	err := r.querier.QueryRow(ctx, query).Scan(&lastTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("unexpected delivery repository get last assigned order time error: %w", err)
	}

	if lastTime == nil {
		return time.Time{}, delivery.ErrDeliveryNotFound
	}

	return *lastTime, nil
}

func (r *Repository) GetCourierIDByOrderID(ctx context.Context, orderID string) (int64, error) {
	query := `
        SELECT courier_id 
        FROM delivery 
        WHERE order_id = $1
    `

	var courierID int64
	err := r.querier.QueryRow(ctx, query, orderID).Scan(&courierID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, delivery.ErrDeliveryNotFound
		}
		return 0, fmt.Errorf("unexpected delivery repository get courier id error: %w", err)
	}

	return courierID, nil
}
