-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE grade_source ADD VALUE IF NOT EXISTS 'assignment';

-- +goose Down
SELECT 1;
