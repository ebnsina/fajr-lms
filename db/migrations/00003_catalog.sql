-- +goose Up
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

CREATE TYPE publish_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE course_visibility AS ENUM ('private', 'unlisted', 'public');
CREATE TYPE lesson_kind AS ENUM ('video', 'audio', 'text', 'pdf', 'link', 'live', 'quiz', 'assignment');

CREATE TABLE courses (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  slug          text NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,126}$'),
  title         text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  summary       text NOT NULL DEFAULT '' CHECK (length(summary) <= 2000),
  dir           text_dir NOT NULL DEFAULT 'auto',
  status        publish_status NOT NULL DEFAULT 'draft',
  visibility    course_visibility NOT NULL DEFAULT 'private',
  price_minor   bigint NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  currency      char(3) NOT NULL DEFAULT 'BDT' CHECK (currency ~ '^[A-Z]{3}$'),
  created_by    uuid REFERENCES users(id) ON DELETE SET NULL,
  published_at  timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug),
  CONSTRAINT published_courses_have_a_date CHECK (status <> 'published' OR published_at IS NOT NULL)
);

CREATE TABLE modules (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id  uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title      text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  -- Fractional position, so a drag reorders one row instead of rewriting the list.
  position   double precision NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE lessons (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  module_id   uuid NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  title       text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  kind        lesson_kind NOT NULL DEFAULT 'text',
  body        text NOT NULL DEFAULT '' CHECK (length(body) <= 200000),
  dir         text_dir NOT NULL DEFAULT 'auto',
  duration_s  integer NOT NULL DEFAULT 0 CHECK (duration_s >= 0),
  is_preview  boolean NOT NULL DEFAULT false,
  status      publish_status NOT NULL DEFAULT 'draft',
  position    double precision NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX courses_tenant_status_idx ON courses (tenant_id, status);
CREATE INDEX modules_course_idx ON modules (course_id, position);
CREATE INDEX lessons_module_idx ON lessons (module_id, position);
CREATE INDEX courses_title_trgm_idx ON courses USING gin (title gin_trgm_ops);

CREATE TRIGGER courses_touch BEFORE UPDATE ON courses FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER modules_touch BEFORE UPDATE ON modules FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER lessons_touch BEFORE UPDATE ON lessons FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE courses ENABLE ROW LEVEL SECURITY;
ALTER TABLE courses FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON courses
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE modules FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON modules
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE lessons ENABLE ROW LEVEL SECURITY;
ALTER TABLE lessons FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON lessons
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON courses, modules, lessons TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS modules;
DROP TABLE IF EXISTS courses;
DROP TYPE IF EXISTS lesson_kind, course_visibility, publish_status;
DROP FUNCTION IF EXISTS touch_updated_at();
-- +goose StatementEnd
