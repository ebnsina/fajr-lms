-- +goose Up
-- +goose StatementBegin

ALTER TABLE orders
  ADD COLUMN refunded_minor bigint NOT NULL DEFAULT 0 CHECK (refunded_minor >= 0),
  ADD COLUMN refund_reason  text NOT NULL DEFAULT '' CHECK (length(refund_reason) <= 500),
  ADD COLUMN refunded_by    uuid REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN refunded_at    timestamptz,
  -- Money can only go back out of an order money came into.
  ADD CONSTRAINT refunds_stay_within_the_order CHECK (refunded_minor <= amount_minor);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders
  DROP CONSTRAINT IF EXISTS refunds_stay_within_the_order,
  DROP COLUMN IF EXISTS refunded_minor,
  DROP COLUMN IF EXISTS refund_reason,
  DROP COLUMN IF EXISTS refunded_by,
  DROP COLUMN IF EXISTS refunded_at;
-- +goose StatementEnd
