-- +goose Up
-- +goose StatementBegin

CREATE TYPE grade_source AS ENUM ('quiz', 'manual');

CREATE TABLE grade_items (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id      uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  quiz_id        uuid UNIQUE REFERENCES quizzes(id) ON DELETE CASCADE,
  source         grade_source NOT NULL,
  title          text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  category       text NOT NULL DEFAULT '' CHECK (length(category) <= 100),
  points_possible integer NOT NULL CHECK (points_possible > 0 AND points_possible <= 100000),
  -- Relative weight within the course; equal weights mean a simple average.
  weight         integer NOT NULL DEFAULT 100 CHECK (weight >= 0 AND weight <= 10000),
  position       double precision NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT quiz_items_reference_a_quiz CHECK ((source = 'quiz') = (quiz_id IS NOT NULL))
);

-- Only scores a teacher entered or changed are stored; quiz scores are read
-- from the attempts themselves, so the two can never fall out of step.
CREATE TABLE grade_overrides (
  grade_item_id uuid NOT NULL REFERENCES grade_items(id) ON DELETE CASCADE,
  enrollment_id uuid NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  points        integer NOT NULL CHECK (points >= 0),
  note          text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
  set_by        uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (grade_item_id, enrollment_id)
);

CREATE INDEX grade_items_course_idx ON grade_items (course_id, position);
CREATE INDEX grade_overrides_enrollment_idx ON grade_overrides (enrollment_id);

CREATE TRIGGER grade_items_touch BEFORE UPDATE ON grade_items FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER grade_overrides_touch BEFORE UPDATE ON grade_overrides FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['grade_items', 'grade_overrides'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS grade_overrides;
DROP TABLE IF EXISTS grade_items;
DROP TYPE IF EXISTS grade_source;
-- +goose StatementEnd
