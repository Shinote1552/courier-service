-- Active: 1763566690944@@127.0.0.1@5432@postgres
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS delivery (
    id          BIGSERIAL PRIMARY KEY,
    courier_id  BIGINT NOT NULL REFERENCES couriers(id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT,
        -- Пока что id couriers должен быть неизменным
        -- Если потребуется - можно будет поменять на UPDATE CASCADE
    order_id    VARCHAR(255) NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deadline    TIMESTAMP NOT NULL
);

CREATE INDEX idx_delivery_courier_id ON delivery USING BTREE (courier_id);
CREATE UNIQUE INDEX idx_delivery_order_unique ON delivery USING BTREE (order_id);

-- для условия deadline < NOW()
CREATE INDEX idx_delivery_deadline ON delivery USING BTREE (deadline); 

CREATE INDEX idx_delivery_created_at ON delivery USING BTREE (created_at DESC); 
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS delivery;
-- +goose StatementEnd