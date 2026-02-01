package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"service/internal/entities"
)

type Delivery struct {
	repository     Repository
	courierService CourierService
	timeFactory    DeliveryTimeFactory
	txManager      TxManager
}

func New(
	repository Repository,
	courierService CourierService,
	timeFactory DeliveryTimeFactory,
	txManager TxManager,
) *Delivery {
	return &Delivery{
		repository:     repository,
		courierService: courierService,
		timeFactory:    timeFactory,
		txManager:      txManager,
	}
}

func (d *Delivery) DeliveryAssign(ctx context.Context, orderID string) (*entities.DeliveryAssignment, error) {
	if !isValidOrderID(orderID) {
		return nil, ErrInvalidOrderID
	}

	deliveryCreatedAt := time.Now().UTC()
	return d.internalDeliveryAssign(ctx, orderID, deliveryCreatedAt)
}

func (d *Delivery) DeliveryUnassign(ctx context.Context, orderID string) (*entities.DeliveryUnassignment, error) {
	if !isValidOrderID(orderID) {
		return nil, ErrInvalidOrderID
	}

	deliveryUnassignment := entities.DeliveryUnassignment{}
	err := d.txManager.Do(ctx, func(ctx context.Context) error {
		courierID, activeDeliveriesCount, err := d.repository.GetCourierIDAndDeliveryCountByOrderIDForAssing(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get courier by order id: %w", err)
		}

		if activeDeliveriesCount > 0 {
			return ErrCourierHasActiveDeliveries
		}

		err = d.repository.Delete(ctx, orderID)
		if err != nil {
			return fmt.Errorf("delete delivery: %w", err)
		}

		newStatus := entities.CourierAvailable
		courierModify := entities.CourierModify{
			ID:     &courierID,
			Status: &newStatus,
		}

		courier, err := d.courierService.UpdateCourier(ctx, courierModify)
		if err != nil {
			return fmt.Errorf("update courier status: %w", err)
		}

		deliveryUnassignment = entities.DeliveryUnassignment{
			CourierID: courier.ID,
			OrderID:   orderID,
			Status:    courier.Status.String(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &deliveryUnassignment, nil
}

func (d *Delivery) CleanupExpiredDeliveries(ctx context.Context) (int64, error) {
	rowsAffected, err := d.repository.UpdateCouriersAvailableWhereDeadlineExpired(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("cleanup timed out: %w", err)
		}
		return 0, fmt.Errorf("cleanup: %w", err)
	}

	return rowsAffected, nil
}

func (d *Delivery) internalDeliveryAssign(ctx context.Context, orderID string, deliveryCreatedAt time.Time) (*entities.DeliveryAssignment, error) {
	if !isValidOrderID(orderID) {
		return nil, ErrInvalidOrderID
	}

	deliveryAssignment := entities.DeliveryAssignment{}

	err := d.txManager.Do(ctx, func(ctx context.Context) error {
		courier, err := d.repository.GetCourierForAssignment(ctx)
		if err != nil {
			return fmt.Errorf("find courier for assignment: %w", err)
		}

		deadline := d.timeFactory.CalculateDeadline(courier.TransportType, time.Now().UTC())
		// по идее Assign часть бизнес логики поэтому время задаем тут а не в БД
		assignTime := time.Now().UTC()

		deliveryModify := entities.DeliveryModify{
			CourierID:  &courier.ID,
			OrderID:    &orderID,
			CreatedAt:  &deliveryCreatedAt,
			AssignedAt: &assignTime,
			Deadline:   &deadline,
		}

		delivery, err := d.repository.Create(ctx, deliveryModify)
		if err != nil {
			return fmt.Errorf("create delivery: %w", err)
		}

		busyStatus := entities.CourierBusy
		courierModify := entities.CourierModify{
			ID:     &courier.ID,
			Status: &busyStatus,
		}

		updatedCourier, err := d.courierService.UpdateCourier(ctx, courierModify)
		if err != nil {
			return fmt.Errorf("update courier status: %w", err)
		}

		deliveryAssignment = entities.DeliveryAssignment{
			CourierID:     updatedCourier.ID,
			OrderID:       delivery.OrderID,
			AssignedAt:    delivery.AssignedAt,
			Deadline:      delivery.Deadline,
			TransportType: updatedCourier.TransportType,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &deliveryAssignment, nil
}

func (d *Delivery) FreeCourierByOrderID(ctx context.Context, orderID string) error {
	if !isValidOrderID(orderID) {
		return ErrInvalidOrderID
	}

	err := d.txManager.Do(ctx, func(ctx context.Context) error {
		courierID, err := d.repository.GetCourierIDByOrderID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get courier by order id: %w", err)
		}

		newStatus := entities.CourierAvailable
		courierModify := entities.CourierModify{
			ID:     &courierID,
			Status: &newStatus,
		}

		updatedCourier, err := d.courierService.UpdateCourier(ctx, courierModify)
		if err != nil || updatedCourier.Status != newStatus {
			return fmt.Errorf("update courier status: %w", err)
		}
		return nil
	})

	return err
}
