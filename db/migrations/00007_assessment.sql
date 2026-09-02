-- +goose Up
-- +goose StatementBegin

CREATE TYPE question_kind AS ENUM ('mcq_single', 'mcq_multi', 'true_false', 'short_answer', 'essay');
CREATE TYPE attempt_state AS ENUM ('in_progress', 'submitted', 'graded', 'expired');

CREATE TABLE quizzes (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  lesson_id     uuid NOT NULL UNIQUE REFERENCES lessons(id) ON DELETE CASCADE,
  title         text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  instructions  text NOT NULL DEFAULT '' CHECK (length(instructions) <= 5000),
  dir           text_dir NOT NULL DEFAULT 'auto',
  time_limit_s  integer NOT NULL DEFAULT 0 CHECK (time_limit_s >= 0),
  max_attempts  smallint NOT NULL DEFAULT 1 CHECK (max_attempts BETWEEN 1 AND 100),
  pass_percent  smallint NOT NULL DEFAULT 50 CHECK (pass_percent BETWEEN 0 AND 100),
  shuffle       boolean NOT NULL DEFAULT false,
  reveal_answers boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE questions (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  quiz_id     uuid NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
  kind        question_kind NOT NULL,
  prompt      text NOT NULL CHECK (length(btrim(prompt)) BETWEEN 1 AND 5000),
  dir         text_dir NOT NULL DEFAULT 'auto',
  points      integer NOT NULL DEFAULT 1 CHECK (points > 0 AND points <= 1000),
  explanation text NOT NULL DEFAULT '' CHECK (length(explanation) <= 5000),
  position    double precision NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE question_options (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  label       text NOT NULL CHECK (length(btrim(label)) BETWEEN 1 AND 2000),
  is_correct  boolean NOT NULL DEFAULT false,
  position    double precision NOT NULL
);

CREATE TABLE quiz_attempts (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  quiz_id       uuid NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
  enrollment_id uuid NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  attempt_no    smallint NOT NULL CHECK (attempt_no > 0),
  state         attempt_state NOT NULL DEFAULT 'in_progress',
  started_at    timestamptz NOT NULL DEFAULT now(),
  expires_at    timestamptz,
  submitted_at  timestamptz,
  graded_at     timestamptz,
  points_awarded integer NOT NULL DEFAULT 0 CHECK (points_awarded >= 0),
  points_possible integer NOT NULL DEFAULT 0 CHECK (points_possible >= 0),
  UNIQUE (quiz_id, user_id, attempt_no)
);

-- Only one attempt may be open at a time, but finished ones are kept forever.
CREATE UNIQUE INDEX quiz_attempts_one_open ON quiz_attempts (quiz_id, user_id)
  WHERE state = 'in_progress';

CREATE TABLE attempt_answers (
  attempt_id     uuid NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
  question_id    uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  option_ids     uuid[] NOT NULL DEFAULT '{}',
  text_answer    text NOT NULL DEFAULT '' CHECK (length(text_answer) <= 20000),
  points_awarded integer,
  needs_grading  boolean NOT NULL DEFAULT false,
  feedback       text NOT NULL DEFAULT '' CHECK (length(feedback) <= 5000),
  graded_by      uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (attempt_id, question_id)
);

CREATE INDEX questions_quiz_idx ON questions (quiz_id, position);
CREATE INDEX question_options_question_idx ON question_options (question_id, position);
CREATE INDEX quiz_attempts_user_idx ON quiz_attempts (user_id, quiz_id);
CREATE INDEX quiz_attempts_grading_idx ON quiz_attempts (tenant_id, state) WHERE state = 'submitted';

CREATE TRIGGER quizzes_touch BEFORE UPDATE ON quizzes FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER questions_touch BEFORE UPDATE ON questions FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER attempt_answers_touch BEFORE UPDATE ON attempt_answers FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['quizzes', 'questions', 'question_options', 'quiz_attempts', 'attempt_answers'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS attempt_answers;
DROP TABLE IF EXISTS quiz_attempts;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS quizzes;
DROP TYPE IF EXISTS attempt_state, question_kind;
-- +goose StatementEnd
