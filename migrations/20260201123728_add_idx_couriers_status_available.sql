-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_couriers_status_available 
ON couriers (status) 
WHERE status = 'available';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_couriers_status_available;
-- +goose StatementEnd
