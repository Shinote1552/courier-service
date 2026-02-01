package delivery

import "service/internal/entities"

func ToDomain(d *DeliveryDB) *entities.Delivery {
	if d == nil {
		return nil
	}
	return &entities.Delivery{
		ID:         d.ID,
		CourierID:  d.CourierID,
		OrderID:    d.OrderID,
		CreatedAt:  d.CreatedAt,
		AssignedAt: d.AssignedAt,
		Deadline:   d.Deadline,
	}
}

func FromDomainModify(d *entities.DeliveryModify) *DeliveryModifyDB {
	if d == nil {
		return nil
	}
	deliveryModifyDB := &DeliveryModifyDB{}

	if d.ID != nil {
		deliveryModifyDB.ID = d.ID
	}
	if d.CourierID != nil {
		deliveryModifyDB.CourierID = d.CourierID
	}
	if d.OrderID != nil {
		deliveryModifyDB.OrderID = d.OrderID
	}
	if d.CreatedAt != nil {
		deliveryModifyDB.CreatedAt = d.CreatedAt
	}
	if d.AssignedAt != nil {
		deliveryModifyDB.AssignedAt = d.AssignedAt
	}
	if d.Deadline != nil {
		deliveryModifyDB.Deadline = d.Deadline
	}

	return deliveryModifyDB
}

func ToCourierDomain(c *AvailableCourierDB) *entities.Courier {
	if c == nil {
		return nil
	}
	return &entities.Courier{
		ID:            c.ID,
		Name:          c.Name,
		Phone:         c.Phone,
		Status:        entities.CourierStatusType(c.Status),
		TransportType: entities.CourierTransportType(c.TransportType),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}
