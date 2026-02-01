-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS couriers (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    phone       TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL
        CHECK (status IN ('available', 'busy', 'paused')),
    created_at  TIMESTAMP DEFAULT now(),
    updated_at  TIMESTAMP DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS couriers;
-- +goose StatementEnd