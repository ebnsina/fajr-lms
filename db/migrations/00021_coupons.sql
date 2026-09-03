-- +goose Up
-- +goose StatementBegin

CREATE TYPE discount_kind AS ENUM ('percent', 'amount');

-- A code a school hands out. It can be tied to one course or left open, capped
-- by a number of uses, a window, or both.
CREATE TABLE coupons (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  code          text NOT NULL CHECK (code ~ '^[A-Z0-9][A-Z0-9-]{2,31}$'),
  kind          discount_kind NOT NULL,
  value         bigint NOT NULL CHECK (value > 0),
  course_id     uuid REFERENCES courses(id) ON DELETE CASCADE,
  max_redemptions int CHECK (max_redemptions IS NULL OR max_redemptions > 0),
  redeemed      int NOT NULL DEFAULT 0 CHECK (redeemed >= 0),
  starts_at     timestamptz,
  ends_at       timestamptz,
  active        boolean NOT NULL DEFAULT true,
  created_by    uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, code),
  CONSTRAINT a_percentage_is_a_percentage CHECK (kind <> 'percent' OR value BETWEEN 1 AND 100),
  CONSTRAINT it_ends_after_it_starts CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX coupons_course_idx ON coupons (tenant_id, course_id);

CREATE TRIGGER coupons_touch BEFORE UPDATE ON coupons FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE coupons ENABLE ROW LEVEL SECURITY;
ALTER TABLE coupons FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON coupons USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());
GRANT SELECT, INSERT, UPDATE, DELETE ON coupons TO fajr_app;

-- What the order would have cost, and what was taken off, so a receipt can say.
ALTER TABLE orders
  ADD COLUMN coupon_id uuid REFERENCES coupons(id) ON DELETE SET NULL,
  ADD COLUMN list_amount_minor bigint,
  ADD COLUMN discount_minor bigint NOT NULL DEFAULT 0 CHECK (discount_minor >= 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders
  DROP COLUMN IF EXISTS coupon_id,
  DROP COLUMN IF EXISTS list_amount_minor,
  DROP COLUMN IF EXISTS discount_minor;
DROP TABLE IF EXISTS coupons;
DROP TYPE IF EXISTS discount_kind;
-- +goose StatementEnd
