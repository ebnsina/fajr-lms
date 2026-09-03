-- +goose Up
-- +goose StatementBegin

-- How many questions to put in front of a learner. Left empty, they get the
-- whole quiz, which is what every existing quiz does.
ALTER TABLE quizzes ADD COLUMN draw_count int CHECK (draw_count IS NULL OR draw_count > 0);

-- The paper one attempt was actually served, in the order it was served in.
-- Without this, a shuffled or drawn quiz could not be resumed or graded.
CREATE TABLE attempt_questions (
  attempt_id  uuid NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
  question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  position    int NOT NULL,
  PRIMARY KEY (attempt_id, question_id)
);

CREATE INDEX attempt_questions_order_idx ON attempt_questions (attempt_id, position);

ALTER TABLE attempt_questions ENABLE ROW LEVEL SECURITY;
ALTER TABLE attempt_questions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON attempt_questions USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());
GRANT SELECT, INSERT, UPDATE, DELETE ON attempt_questions TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS attempt_questions;
ALTER TABLE quizzes DROP COLUMN IF EXISTS draw_count;
-- +goose StatementEnd
