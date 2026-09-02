-- +goose Up
-- +goose StatementBegin

CREATE TYPE order_status AS ENUM ('pending', 'awaiting_review', 'paid', 'rejected', 'cancelled', 'refunded');

CREATE TABLE orders (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_id     uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  provider      text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_]{1,30}$'),
  amount_minor  bigint NOT NULL CHECK (amount_minor > 0),
  currency      char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  status        order_status NOT NULL DEFAULT 'pending',
  -- Shown to the payer and quoted on a deposit slip, so it must be unique.
  reference     text NOT NULL CHECK (reference ~ '^[A-Z0-9-]{6,32}$'),
  provider_ref  text NOT NULL DEFAULT '' CHECK (length(provider_ref) <= 255),
  note          text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
  proof_media_id uuid REFERENCES media_assets(id) ON DELETE SET NULL,
  reviewed_by   uuid REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at   timestamptz,
  paid_at       timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, reference),
  CONSTRAINT paid_orders_have_a_date CHECK (status <> 'paid' OR paid_at IS NOT NULL)
);

-- One open order per learner per course; settled ones are kept for the record.
CREATE UNIQUE INDEX orders_one_open_per_course ON orders (course_id, user_id)
  WHERE status IN ('pending', 'awaiting_review');

CREATE INDEX orders_tenant_status_idx ON orders (tenant_id, status, created_at DESC);
CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);

-- Append-only, so a replayed webhook is recorded once and reconciliation has a
-- trail even when a gateway sends the same event three times.
CREATE TABLE payment_events (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_id    uuid REFERENCES orders(id) ON DELETE SET NULL,
  provider    text NOT NULL,
  event_id    text NOT NULL CHECK (length(event_id) BETWEEN 1 AND 255),
  kind        text NOT NULL CHECK (length(kind) BETWEEN 1 AND 64),
  payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, event_id)
);

CREATE INDEX payment_events_order_idx ON payment_events (order_id, created_at DESC);

CREATE TRIGGER orders_touch BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON orders
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE payment_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payment_events
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON orders TO fajr_app;
GRANT SELECT, INSERT ON payment_events TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;
-- +goose StatementEnd
