-- +goose Up
-- +goose StatementBegin

-- The back office: us, looking at every school. It reads across tenants by
-- design, so it does not get a blanket bypass — every view it has is one named
-- function, and every action it takes is written down.
CREATE TABLE platform_staff (
  user_id       uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  role          text NOT NULL DEFAULT 'owner' CHECK (role IN ('owner', 'support')),
  password_hash text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at  timestamptz
);

CREATE TABLE staff_audit (
  id         bigserial PRIMARY KEY,
  user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  action     text NOT NULL CHECK (length(action) <= 120),
  subject    text NOT NULL DEFAULT '' CHECK (length(subject) <= 200),
  detail     text NOT NULL DEFAULT '' CHECK (length(detail) <= 2000),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX staff_audit_created_idx ON staff_audit (created_at DESC);

ALTER TABLE platform_staff ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_staff FORCE ROW LEVEL SECURITY;
ALTER TABLE staff_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_audit FORCE ROW LEVEL SECURITY;

-- A lead is worked, so it carries where it got to.
ALTER TABLE demo_leads ADD COLUMN state text NOT NULL DEFAULT 'new'
  CHECK (state IN ('new', 'contacted', 'qualified', 'won', 'lost'));
ALTER TABLE demo_leads ADD COLUMN worked_note text NOT NULL DEFAULT ''
  CHECK (length(worked_note) <= 2000);

-- Signing in as staff: the password lives here and nowhere else.
CREATE FUNCTION staff_by_email(p_email text) RETURNS SETOF platform_staff
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT s.* FROM platform_staff s JOIN users u ON u.id = s.user_id
  WHERE u.email = lower(p_email)
$fn$;

CREATE FUNCTION staff_role(p_user_id uuid) RETURNS SETOF text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT role FROM platform_staff WHERE user_id = p_user_id
$fn$;

CREATE FUNCTION staff_seen(p_user_id uuid) RETURNS void
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $fn$
  UPDATE platform_staff SET last_seen_at = now() WHERE user_id = p_user_id
$fn$;

CREATE FUNCTION staff_log(p_user_id uuid, p_action text, p_subject text, p_detail text)
RETURNS void
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $fn$
  INSERT INTO staff_audit (user_id, action, subject, detail)
  VALUES (p_user_id, p_action, left(p_subject, 200), left(p_detail, 2000))
$fn$;

-- The four views the back office has. Each one is a whole-platform read, which
-- is why each one is named, audited at the handler, and no wider than it needs.
CREATE FUNCTION admin_leads(p_state text, p_query text, p_limit int, p_offset int)
RETURNS SETOF demo_leads
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT * FROM demo_leads
  WHERE (nullif(p_state, '') IS NULL OR state = p_state)
    AND (nullif(p_query, '') IS NULL
         OR full_name ILIKE '%' || p_query || '%'
         OR email ILIKE '%' || p_query || '%'
         OR organisation ILIKE '%' || p_query || '%')
  ORDER BY created_at DESC
  LIMIT least(coalesce(p_limit, 50), 500) OFFSET greatest(coalesce(p_offset, 0), 0)
$fn$;

CREATE FUNCTION admin_set_lead(p_id uuid, p_state text, p_note text) RETURNS SETOF demo_leads
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $fn$
  UPDATE demo_leads SET state = p_state, worked_note = left(p_note, 2000)
  WHERE id = p_id RETURNING *
$fn$;

CREATE FUNCTION admin_tenants(p_query text, p_limit int, p_offset int) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT coalesce(jsonb_agg(row), '[]'::jsonb) FROM (
    SELECT jsonb_build_object(
      'id', t.id, 'slug', t.slug, 'name', t.name, 'kind', t.kind, 'status', t.status,
      'demo', t.demo, 'created_at', t.created_at,
      'members', (SELECT count(*) FROM memberships m WHERE m.tenant_id = t.id),
      'courses', (SELECT count(*) FROM courses c WHERE c.tenant_id = t.id),
      'learners', (SELECT count(DISTINCT e.user_id) FROM enrollments e WHERE e.tenant_id = t.id),
      'certificates', (SELECT count(*) FROM certificates ct WHERE ct.tenant_id = t.id),
      'orders', (SELECT count(*) FROM orders o WHERE o.tenant_id = t.id AND o.status = 'paid'),
      'last_activity', greatest(t.updated_at,
        (SELECT max(c.updated_at) FROM courses c WHERE c.tenant_id = t.id),
        (SELECT max(e.updated_at) FROM enrollments e WHERE e.tenant_id = t.id))
    ) AS row
    FROM tenants t
    WHERE nullif(p_query, '') IS NULL
       OR t.name ILIKE '%' || p_query || '%' OR t.slug ILIKE '%' || p_query || '%'
    ORDER BY t.created_at DESC
    LIMIT least(coalesce(p_limit, 50), 500) OFFSET greatest(coalesce(p_offset, 0), 0)
  ) rows
$fn$;

CREATE FUNCTION admin_tenant(p_tenant_id uuid) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT jsonb_build_object(
    'tenant', to_jsonb(t),
    'members', coalesce((
      SELECT jsonb_agg(jsonb_build_object(
        'user_id', u.id, 'full_name', u.full_name,
        'contact', coalesce(u.email, u.phone, ''), 'role', m.role, 'since', m.created_at
      ) ORDER BY m.created_at)
      FROM memberships m JOIN users u ON u.id = m.user_id WHERE m.tenant_id = t.id
    ), '[]'::jsonb),
    'courses', coalesce((
      SELECT jsonb_agg(jsonb_build_object(
        'title', c.title, 'status', c.status, 'learners',
        (SELECT count(*) FROM enrollments e WHERE e.course_id = c.id)
      ) ORDER BY c.created_at)
      FROM courses c WHERE c.tenant_id = t.id
    ), '[]'::jsonb),
    'orders', coalesce((
      SELECT jsonb_agg(jsonb_build_object(
        'reference', o.reference, 'status', o.status, 'provider', o.provider,
        'amount_minor', o.amount_minor, 'currency', o.currency, 'created_at', o.created_at
      ) ORDER BY o.created_at DESC)
      FROM orders o WHERE o.tenant_id = t.id
    ), '[]'::jsonb),
    'certificates', (SELECT count(*) FROM certificates ct WHERE ct.tenant_id = t.id)
  )
  FROM tenants t WHERE t.id = p_tenant_id
$fn$;

-- The weekly numbers, as one row. Every one of them is a count you can check
-- by hand, which is the point.
CREATE FUNCTION admin_overview() RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $fn$
  SELECT jsonb_build_object(
    'schools', (SELECT count(*) FROM tenants WHERE NOT demo),
    'demo_schools', (SELECT count(*) FROM tenants WHERE demo),
    'people', (SELECT count(*) FROM users),
    'leads', (SELECT count(*) FROM demo_leads),
    'leads_this_week', (SELECT count(*) FROM demo_leads WHERE created_at > now() - interval '7 days'),
    'leads_won', (SELECT count(*) FROM demo_leads WHERE state = 'won'),
    -- A lead that came back and opened a school of its own, which is the only
    -- number that says whether the demo is doing its job.
    'leads_converted', (SELECT count(*) FROM demo_leads l WHERE EXISTS (
      SELECT 1 FROM users u JOIN memberships m ON m.user_id = u.id
      JOIN tenants t ON t.id = m.tenant_id
      WHERE u.email = l.email AND m.role = 'owner' AND NOT t.demo)),
    'courses', (SELECT count(*) FROM courses),
    'enrollments', (SELECT count(*) FROM enrollments),
    'certificates', (SELECT count(*) FROM certificates),
    'paid_orders', (SELECT count(*) FROM orders WHERE status = 'paid'),
    'paid_minor', (SELECT coalesce(sum(amount_minor), 0) FROM orders WHERE status = 'paid')
  )
$fn$;

REVOKE ALL ON FUNCTION staff_by_email(text), staff_role(uuid), staff_seen(uuid),
  staff_log(uuid, text, text, text), admin_leads(text, text, int, int),
  admin_set_lead(uuid, text, text), admin_tenants(text, int, int),
  admin_tenant(uuid), admin_overview() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION staff_by_email(text), staff_role(uuid), staff_seen(uuid),
  staff_log(uuid, text, text, text), admin_leads(text, text, int, int),
  admin_set_lead(uuid, text, text), admin_tenants(text, int, int),
  admin_tenant(uuid), admin_overview() TO fajr_app;

-- The one staff account. Change the password from the back office once you are in.
INSERT INTO users (email, full_name)
VALUES ('sina@fajrlabs.com', 'Sina')
ON CONFLICT (email) DO NOTHING;

INSERT INTO platform_staff (user_id, role, password_hash)
SELECT id, 'owner', '$2a$12$heFvI0Y2Jd39iiC/ffvrn.ueCT0eEC1vapVcB/onN.N8wFesMYDrC'
FROM users WHERE email = 'sina@fajrlabs.com'
ON CONFLICT (user_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS admin_overview(), admin_tenant(uuid), admin_tenants(text, int, int),
  admin_set_lead(uuid, text, text), admin_leads(text, text, int, int),
  staff_log(uuid, text, text, text), staff_seen(uuid), staff_role(uuid), staff_by_email(text);
ALTER TABLE demo_leads DROP COLUMN IF EXISTS worked_note;
ALTER TABLE demo_leads DROP COLUMN IF EXISTS state;
DROP TABLE IF EXISTS staff_audit;
DROP TABLE IF EXISTS platform_staff;
-- +goose StatementEnd
