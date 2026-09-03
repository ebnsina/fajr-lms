-- +goose Up
-- +goose StatementBegin

-- A course can be sold in parts. One part means paid in full, which is what
-- every course does today.
ALTER TABLE courses
  ADD COLUMN installments smallint NOT NULL DEFAULT 1
    CHECK (installments BETWEEN 1 AND 24),
  ADD COLUMN installment_gap_days smallint NOT NULL DEFAULT 30
    CHECK (installment_gap_days BETWEEN 1 AND 365);

-- One learner's agreement to pay a course off over time. The orders are made
-- one at a time, so the payment machinery does not change.
CREATE TABLE payment_plans (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_id    uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  total_minor  bigint NOT NULL CHECK (total_minor > 0),
  currency     char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  parts        smallint NOT NULL CHECK (parts BETWEEN 2 AND 24),
  paid_parts   smallint NOT NULL DEFAULT 0 CHECK (paid_parts >= 0),
  gap_days     smallint NOT NULL CHECK (gap_days BETWEEN 1 AND 365),
  next_due_on  date,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT parts_paid_stay_within_the_plan CHECK (paid_parts <= parts),
  UNIQUE (course_id, user_id)
);

CREATE INDEX payment_plans_due_idx ON payment_plans (tenant_id, next_due_on)
  WHERE next_due_on IS NOT NULL;

ALTER TABLE orders
  ADD COLUMN plan_id uuid REFERENCES payment_plans(id) ON DELETE SET NULL,
  ADD COLUMN part_no smallint CHECK (part_no IS NULL OR part_no > 0);

CREATE TRIGGER payment_plans_touch BEFORE UPDATE ON payment_plans
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE payment_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_plans FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payment_plans
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());
GRANT SELECT, INSERT, UPDATE, DELETE ON payment_plans TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP COLUMN IF EXISTS plan_id, DROP COLUMN IF EXISTS part_no;
DROP TABLE IF EXISTS payment_plans;
ALTER TABLE courses
  DROP COLUMN IF EXISTS installments,
  DROP COLUMN IF EXISTS installment_gap_days;
-- +goose StatementEnd
