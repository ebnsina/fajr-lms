-- +goose Up
-- +goose StatementBegin

CREATE TYPE attendance_status AS ENUM ('present', 'late', 'absent', 'excused');

CREATE TABLE class_sessions (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id  uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title      text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  location   text NOT NULL DEFAULT '' CHECK (length(location) <= 200),
  starts_at  timestamptz NOT NULL,
  ends_at    timestamptz,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT session_ends_after_it_starts CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE TABLE attendance (
  session_id    uuid NOT NULL REFERENCES class_sessions(id) ON DELETE CASCADE,
  enrollment_id uuid NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  status        attendance_status NOT NULL,
  note          text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
  marked_by     uuid REFERENCES users(id) ON DELETE SET NULL,
  marked_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (session_id, enrollment_id)
);

-- A guardian is a user who may see another user's attendance and results.
CREATE TABLE guardianships (
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  guardian_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  student_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  relation   text NOT NULL DEFAULT '' CHECK (length(relation) <= 50),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, guardian_id, student_id),
  CONSTRAINT a_guardian_is_not_the_student CHECK (guardian_id <> student_id)
);

CREATE INDEX class_sessions_course_idx ON class_sessions (course_id, starts_at DESC);
CREATE INDEX attendance_enrollment_idx ON attendance (enrollment_id);
CREATE INDEX guardianships_student_idx ON guardianships (tenant_id, student_id);

CREATE TRIGGER class_sessions_touch BEFORE UPDATE ON class_sessions FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['class_sessions', 'attendance', 'guardianships'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS guardianships;
DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS class_sessions;
DROP TYPE IF EXISTS attendance_status;
-- +goose StatementEnd
