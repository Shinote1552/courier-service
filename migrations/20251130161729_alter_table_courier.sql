-- +goose Up
-- +goose StatementBegin
ALTER TABLE couriers
ADD COLUMN IF NOT EXISTS transport_type TEXT NOT NULL DEFAULT 'on_foot'
    CHECK (transport_type IN ('on_foot', 'scooter', 'car'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE couriers
DROP COLUMN IF EXISTS transport_type;
-- +goose StatementEnd
