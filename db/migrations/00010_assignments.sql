-- +goose Up
-- +goose StatementBegin

CREATE TYPE submission_state AS ENUM ('draft', 'submitted', 'returned');

CREATE TABLE assignments (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  lesson_id     uuid NOT NULL UNIQUE REFERENCES lessons(id) ON DELETE CASCADE,
  title         text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  instructions  text NOT NULL DEFAULT '' CHECK (length(instructions) <= 20000),
  dir           text_dir NOT NULL DEFAULT 'auto',
  points        integer NOT NULL DEFAULT 100 CHECK (points > 0 AND points <= 100000),
  due_at        timestamptz,
  allow_late    boolean NOT NULL DEFAULT true,
  -- Percent removed from a late submission's mark, applied once at grading.
  late_penalty  smallint NOT NULL DEFAULT 0 CHECK (late_penalty BETWEEN 0 AND 100),
  max_files     smallint NOT NULL DEFAULT 5 CHECK (max_files BETWEEN 0 AND 20),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  assignment_id  uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  enrollment_id  uuid NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  state          submission_state NOT NULL DEFAULT 'draft',
  body           text NOT NULL DEFAULT '' CHECK (length(body) <= 100000),
  media_ids      uuid[] NOT NULL DEFAULT '{}',
  is_late        boolean NOT NULL DEFAULT false,
  submitted_at   timestamptz,
  points_awarded integer CHECK (points_awarded >= 0),
  feedback       text NOT NULL DEFAULT '' CHECK (length(feedback) <= 10000),
  graded_by      uuid REFERENCES users(id) ON DELETE SET NULL,
  graded_at      timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (assignment_id, enrollment_id),
  CONSTRAINT submitted_work_has_a_date CHECK (state = 'draft' OR submitted_at IS NOT NULL)
);

ALTER TABLE grade_items ADD COLUMN assignment_id uuid UNIQUE REFERENCES assignments(id) ON DELETE CASCADE;
ALTER TABLE grade_items DROP CONSTRAINT quiz_items_reference_a_quiz;
ALTER TABLE grade_items ADD CONSTRAINT items_reference_their_source CHECK (
  (source = 'quiz') = (quiz_id IS NOT NULL) AND (source = 'assignment') = (assignment_id IS NOT NULL)
);

CREATE INDEX submissions_assignment_idx ON submissions (assignment_id, state);
CREATE INDEX submissions_grading_idx ON submissions (tenant_id, state) WHERE state = 'submitted';

CREATE TRIGGER assignments_touch BEFORE UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER submissions_touch BEFORE UPDATE ON submissions FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['assignments', 'submissions'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE grade_items DROP COLUMN IF EXISTS assignment_id;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS assignments;
DROP TYPE IF EXISTS submission_state;
-- +goose StatementEnd
