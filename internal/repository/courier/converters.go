package courier

import (
	"service/internal/entities"
)

func ToDomain(c *CourierDB) *entities.Courier {
	if c == nil {
		return nil
	}

	statusType := entities.CourierStatusType(c.Status)
	return &entities.Courier{
		ID:            c.ID,
		Name:          c.Name,
		Phone:         c.Phone,
		Status:        statusType,
		TransportType: entities.CourierTransportType(c.TransportType),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func FromDomainModify(courierModify *entities.CourierModify) *CourierModifyDB {
	if courierModify == nil {
		return nil
	}
	courierDB := &CourierModifyDB{}

	if courierModify.ID != nil {
		courierDB.ID = courierModify.ID
	}
	if courierModify.Name != nil {
		courierDB.Name = courierModify.Name
	}
	if courierModify.Phone != nil {
		courierDB.Phone = courierModify.Phone
	}
	if courierModify.Status != nil {
		statusType := courierModify.Status.String()
		courierDB.Status = &statusType
	}
	if courierModify.TransportType != nil {
		transportType := courierModify.TransportType.String()
		courierDB.TransportType = &transportType
	}

	return courierDB
}

func ToDomainList(couriersDB []CourierDB) []entities.Courier {
	if len(couriersDB) == 0 {
		return []entities.Courier{}
	}

	result := make([]entities.Courier, len(couriersDB))
	for i, courierDB := range couriersDB {
		result[i] = *ToDomain(&courierDB)
	}
	return result
}
