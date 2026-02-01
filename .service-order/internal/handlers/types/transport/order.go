package transport

type Item struct {
	FoodID   string `json:"food_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

type DeliveryAddress struct {
	Street    string `json:"street,omitempty"`
	House     string `json:"house,omitempty"`
	Apartment string `json:"apartment,omitempty"`
	Floor     string `json:"floor,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

type CreateOrderRequest struct {
	OrderNumber  string          `json:"order_number,omitempty"`
	FIO          string          `json:"fio,omitempty"`
	RestaurantID string          `json:"restaurant_id"`
	Items        []Item          `json:"items"`
	TotalPrice   int64           `json:"total_price"`
	Address      DeliveryAddress `json:"address"`
}

type UpdateOrderRequest struct {
	OrderNumber *string          `json:"order_number,omitempty"`
	FIO         *string          `json:"fio,omitempty"`
	Items       *[]Item          `json:"items,omitempty"`
	TotalPrice  *int64           `json:"total_price,omitempty"`
	Address     *DeliveryAddress `json:"address,omitempty"`
}

type OrderResponse struct {
	ID                string          `json:"id"`
	UserID            string          `json:"user_id"`
	OrderNumber       string          `json:"order_number,omitempty"`
	FIO               string          `json:"fio,omitempty"`
	RestaurantID      string          `json:"restaurant_id"`
	Items             []Item          `json:"items"`
	TotalPrice        int64           `json:"total_price"`
	Address           DeliveryAddress `json:"address"`
	Status            string          `json:"status"`
	CreatedAt         string          `json:"created_at"`
	UpdatedAt         string          `json:"updated_at"`
	EstimatedDelivery string          `json:"estimated_delivery"`
}

type DeleteOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
