-- +goose Up
-- +goose StatementBegin

CREATE TABLE certificates (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id     uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  enrollment_id uuid NOT NULL UNIQUE REFERENCES enrollments(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- Quotable in a phone call or written on a form.
  serial        text NOT NULL UNIQUE CHECK (serial ~ '^FJR-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$'),
  -- Names and titles are copied, not joined: a certificate must keep saying
  -- what it said on the day it was issued, whatever is renamed afterwards.
  recipient_name text NOT NULL CHECK (length(btrim(recipient_name)) BETWEEN 1 AND 200),
  course_title   text NOT NULL CHECK (length(btrim(course_title)) BETWEEN 1 AND 200),
  issuer_name    text NOT NULL CHECK (length(btrim(issuer_name)) BETWEEN 1 AND 200),
  grade_percent  smallint CHECK (grade_percent BETWEEN 0 AND 100),
  issued_at      timestamptz NOT NULL DEFAULT now(),
  revoked_at     timestamptz,
  revoked_reason text NOT NULL DEFAULT '' CHECK (length(revoked_reason) <= 500),
  issued_by      uuid REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX certificates_user_idx ON certificates (tenant_id, user_id, issued_at DESC);

ALTER TABLE certificates ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificates FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON certificates
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON certificates TO fajr_app;

-- Verification is public and happens before any tenant is known, so it goes
-- through one narrow function that exposes nothing else.
CREATE VIEW certificate_verifications WITH (security_invoker = false) AS
  SELECT c.serial, c.recipient_name, c.course_title, c.issuer_name,
         c.grade_percent, c.issued_at, c.revoked_at, t.default_dir AS tenant_dir
  FROM certificates c JOIN tenants t ON t.id = c.tenant_id;

GRANT SELECT ON certificate_verifications TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS certificate_verifications;
DROP TABLE IF EXISTS certificates;
-- +goose StatementEnd
