package courier

import (
	"context"
	"fmt"

	"service/internal/entities"
)

type Courier struct {
	repository Repository
	txManager  TxManager
}

func New(repository Repository, txManager TxManager) *Courier {
	return &Courier{
		repository: repository,
		txManager:  txManager,
	}
}

func (s *Courier) CreateCourier(ctx context.Context, courierModify entities.CourierModify) (int64, error) {
	if courierModify.Name == nil ||
		courierModify.Phone == nil ||
		courierModify.Status == nil ||
		courierModify.TransportType == nil {
		return 0, ErrMissingRequiredFields
	}

	if !isValidName(*courierModify.Name) {
		return 0, ErrInvalidName
	}
	if !isValidPhone(*courierModify.Phone) {
		return 0, ErrInvalidPhone
	}
	if !isValidStatus(courierModify.Status.String()) {
		return 0, ErrInvalidStatus
	}
	if !isValidTransport(courierModify.TransportType.String()) {
		return 0, ErrInvalidTransport
	}

	id, err := s.repository.Create(ctx, courierModify)
	if err != nil {
		return 0, fmt.Errorf("create courier: %w", err)
	}

	return id, nil
}

func (s *Courier) UpdateCourier(ctx context.Context, courierModify entities.CourierModify) (*entities.Courier, error) {
	if courierModify.Name == nil &&
		courierModify.Phone == nil &&
		courierModify.Status == nil &&
		courierModify.TransportType == nil {
		return nil, fmt.Errorf("no fields to update: %w", ErrMissingRequiredFields)
	}

	if courierModify.Name != nil && !isValidName(*courierModify.Name) {
		return nil, ErrInvalidName
	}
	if courierModify.Phone != nil && !isValidPhone(*courierModify.Phone) {
		return nil, ErrInvalidPhone
	}
	if courierModify.Status != nil && !isValidStatus(courierModify.Status.String()) {
		return nil, ErrInvalidStatus
	}
	if courierModify.TransportType != nil && !isValidTransport(courierModify.TransportType.String()) {
		return nil, ErrInvalidTransport
	}

	courier, err := s.repository.Update(ctx, courierModify)
	if err != nil {
		return nil, fmt.Errorf("failed to update courier: %w", err)
	}
	return courier, nil
}

func (s *Courier) GetCourier(ctx context.Context, id int64) (*entities.Courier, error) {
	courier, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier: %w", err)
	}

	return courier, nil
}

func (s *Courier) GetCouriers(ctx context.Context) ([]entities.Courier, error) {
	couriers, err := s.repository.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get couriers: %w", err)
	}

	return couriers, nil
}
