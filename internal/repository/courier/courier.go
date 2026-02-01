package courier

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"service/internal/entities"
	"service/internal/repository"
	"service/internal/service/courier"
)

var qb sq.StatementBuilderType = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Repository struct {
	querier Querier
}

func New(querier Querier) *Repository {
	return &Repository{
		querier: querier,
	}
}

func (r *Repository) Create(ctx context.Context, courierModifyEntity entities.CourierModify) (int64, error) {
	courierModifyModel := FromDomainModify(&courierModifyEntity)
	query := `INSERT INTO couriers (name, phone, status, transport_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.querier.QueryRow(
		ctx,
		query,
		courierModifyModel.Name,
		courierModifyModel.Phone,
		courierModifyModel.Status,
		courierModifyModel.TransportType,
	).Scan(&id)
	if err != nil {
		if repository.IsPgErrorWithCode(err, repository.PgErrUniqueViolation) {
			return 0, courier.ErrConflict
		}
		return 0, fmt.Errorf("unexpected courier repository create error: %w", err)
	}

	return id, nil
}

func (r *Repository) Update(ctx context.Context, courierModifyEntity entities.CourierModify) (*entities.Courier, error) {
	courierModifyModel := FromDomainModify(&courierModifyEntity)

	builder := qb.
		Update("couriers")

	// опционнные поля
	if courierModifyModel.Name != nil {
		builder = builder.Set("name", courierModifyModel.Name)
	}
	if courierModifyModel.Phone != nil {
		builder = builder.Set("phone", courierModifyModel.Phone)
	}
	if courierModifyModel.Status != nil {
		builder = builder.Set("status", courierModifyModel.Status)
	}
	if courierModifyModel.TransportType != nil {
		builder = builder.Set("transport_type", courierModifyModel.TransportType)
	}

	builder = builder.Set("updated_at", sq.Expr("NOW()"))

	builder = builder.
		Where(sq.Eq{"ID": courierModifyModel.ID}).
		Suffix("RETURNING ID, name, phone, status, transport_type, created_at, updated_at")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("unexpected courier repository update error: %w", err)
	}

	var courierModel CourierDB
	err = r.querier.QueryRow(ctx, query, args...).
		Scan(
			&courierModel.ID,
			&courierModel.Name,
			&courierModel.Phone,
			&courierModel.Status,
			&courierModel.TransportType,
			&courierModel.CreatedAt,
			&courierModel.UpdatedAt,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, courier.ErrCourierNotFound
		}

		if repository.IsPgErrorWithCode(err, repository.PgErrUniqueViolation) {
			return nil, courier.ErrConflict
		}

		return nil, fmt.Errorf("unexpected courier repository update error: %w", err)
	}

	return ToDomain(&courierModel), nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.Courier, error) {
	query := `SELECT id, name, phone, status, transport_type, created_at, updated_at
		FROM couriers
		WHERE id = $1`

	var courierModel CourierDB
	err := r.querier.QueryRow(ctx, query, id).
		Scan(
			&courierModel.ID,
			&courierModel.Name,
			&courierModel.Phone,
			&courierModel.Status,
			&courierModel.TransportType,
			&courierModel.CreatedAt,
			&courierModel.UpdatedAt,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, courier.ErrCourierNotFound
		}

		return nil, fmt.Errorf("unexpected courier repository getbyid error: %w", err)
	}

	return ToDomain(&courierModel), nil
}

func (r *Repository) GetAll(ctx context.Context) ([]entities.Courier, error) {
	query := `
	SELECT id, name, phone, status, transport_type, created_at, updated_at
	FROM couriers
	ORDER BY id`

	rows, err := r.querier.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unexpected courier repository getall error: %w", err)
	}
	defer rows.Close()

	// начальная емкость, getall может вернуть очень много
	// так и мало, не знаю какая золотая середина
	CourierModels := make([]CourierDB, 0, 8)
	for rows.Next() {
		var courierModel CourierDB
		err := rows.Scan(
			&courierModel.ID,
			&courierModel.Name,
			&courierModel.Phone,
			&courierModel.Status,
			&courierModel.TransportType,
			&courierModel.CreatedAt,
			&courierModel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("unexpected courier repository getall error: %w", err)
		}
		CourierModels = append(CourierModels, courierModel)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("unexpected courier repository getall error: %w", err)
	}

	return ToDomainList(CourierModels), nil
}
