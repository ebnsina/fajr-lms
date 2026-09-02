-- +goose Up
-- +goose StatementBegin

CREATE TYPE enrollment_status AS ENUM ('active', 'completed', 'cancelled');
CREATE TYPE enrollment_source AS ENUM ('self', 'staff', 'purchase', 'import');
CREATE TYPE progress_state AS ENUM ('not_started', 'in_progress', 'completed');

CREATE TABLE enrollments (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id    uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status       enrollment_status NOT NULL DEFAULT 'active',
  source       enrollment_source NOT NULL DEFAULT 'staff',
  completed_at timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (course_id, user_id),
  CONSTRAINT completed_enrollments_have_a_date CHECK (status <> 'completed' OR completed_at IS NOT NULL)
);

CREATE TABLE lesson_progress (
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  enrollment_id uuid NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  lesson_id     uuid NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  state         progress_state NOT NULL DEFAULT 'in_progress',
  -- Resume point in seconds. Merged with GREATEST so a stale offline sync
  -- cannot rewind a learner who has since watched further.
  position_s    integer NOT NULL DEFAULT 0 CHECK (position_s >= 0),
  completed_at  timestamptz,
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (enrollment_id, lesson_id)
);

CREATE INDEX enrollments_user_idx ON enrollments (user_id, status);
CREATE INDEX enrollments_course_idx ON enrollments (course_id, status);
CREATE INDEX lesson_progress_lesson_idx ON lesson_progress (lesson_id);

CREATE TRIGGER enrollments_touch BEFORE UPDATE ON enrollments FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE enrollments ENABLE ROW LEVEL SECURITY;
ALTER TABLE enrollments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON enrollments
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE lesson_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE lesson_progress FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON lesson_progress
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON enrollments, lesson_progress TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lesson_progress;
DROP TABLE IF EXISTS enrollments;
DROP TYPE IF EXISTS progress_state, enrollment_source, enrollment_status;
-- +goose StatementEnd
