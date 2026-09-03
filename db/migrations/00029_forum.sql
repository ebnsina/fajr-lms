-- +goose Up
-- +goose StatementBegin

-- Discussion attached to a course: a question asked once, answered once, and
-- read by everybody else who was about to ask it.
CREATE TABLE forum_threads (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id    uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title        text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  dir          text_dir NOT NULL DEFAULT 'auto',
  author_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  pinned       boolean NOT NULL DEFAULT false,
  locked       boolean NOT NULL DEFAULT false,
  reply_count  integer NOT NULL DEFAULT 0 CHECK (reply_count >= 0),
  last_post_at timestamptz NOT NULL DEFAULT now(),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE forum_posts (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  thread_id  uuid NOT NULL REFERENCES forum_threads(id) ON DELETE CASCADE,
  author_id  uuid REFERENCES users(id) ON DELETE SET NULL,
  body       text NOT NULL CHECK (length(btrim(body)) BETWEEN 1 AND 10000),
  dir        text_dir NOT NULL DEFAULT 'auto',
  -- A removed post keeps its place so the thread still reads in order.
  removed_at timestamptz,
  removed_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX forum_threads_course_idx ON forum_threads (course_id, pinned DESC, last_post_at DESC);
CREATE INDEX forum_posts_thread_idx ON forum_posts (thread_id, created_at);

CREATE TRIGGER forum_threads_touch BEFORE UPDATE ON forum_threads FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER forum_posts_touch BEFORE UPDATE ON forum_posts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE forum_threads ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_threads FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON forum_threads
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE forum_posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_posts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON forum_posts
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON forum_threads TO fajr_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON forum_posts TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS forum_posts;
DROP TABLE IF EXISTS forum_threads;
-- +goose StatementEnd
