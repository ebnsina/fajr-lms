-- +goose Up
-- +goose StatementBegin

-- A demo school is a real tenant that nobody may change: whoever asks for a
-- demo is signed into one and can open every screen, but every write is
-- refused at the door.
ALTER TABLE tenants ADD COLUMN demo boolean NOT NULL DEFAULT false;

-- What somebody told us about themselves on the way in. No tenant owns this:
-- it is ours, and it is the only reason the demo asks anything at all.
CREATE TABLE demo_leads (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name    text NOT NULL CHECK (length(full_name) BETWEEN 1 AND 200),
  email        text NOT NULL CHECK (length(email) BETWEEN 3 AND 320),
  phone        text NOT NULL DEFAULT '' CHECK (length(phone) <= 40),
  organisation text NOT NULL DEFAULT '' CHECK (length(organisation) <= 200),
  role         text NOT NULL DEFAULT '' CHECK (length(role) <= 120),
  learners     text NOT NULL DEFAULT '' CHECK (length(learners) <= 40),
  runs         text NOT NULL DEFAULT '' CHECK (length(runs) <= 40),
  note         text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX demo_leads_created_idx ON demo_leads (created_at DESC);
CREATE INDEX demo_leads_email_idx ON demo_leads (lower(email));

-- Locked to the app role: a lead is written through the function below and
-- read back by nobody but us, straight from the database.
ALTER TABLE demo_leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE demo_leads FORCE ROW LEVEL SECURITY;

CREATE FUNCTION record_demo_lead(
  p_full_name text, p_email text, p_phone text, p_organisation text,
  p_role text, p_learners text, p_runs text, p_note text
) RETURNS uuid
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  INSERT INTO demo_leads (full_name, email, phone, organisation, role, learners, runs, note)
  VALUES (p_full_name, lower(p_email), p_phone, p_organisation, p_role, p_learners, p_runs, p_note)
  RETURNING id
$$;

REVOKE ALL ON FUNCTION record_demo_lead(text, text, text, text, text, text, text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION record_demo_lead(text, text, text, text, text, text, text, text) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS record_demo_lead(text, text, text, text, text, text, text, text);
DROP TABLE IF EXISTS demo_leads;
ALTER TABLE tenants DROP COLUMN IF EXISTS demo;
-- +goose StatementEnd
