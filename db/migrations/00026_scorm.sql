-- +goose Up
-- +goose StatementBegin

-- A course package built somewhere else. One per lesson, so a SCORM lesson is
-- an ordinary lesson with a package attached.
CREATE TABLE scorm_packages (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  lesson_id   uuid NOT NULL UNIQUE REFERENCES lessons(id) ON DELETE CASCADE,
  title       text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 300),
  entry_href  text NOT NULL CHECK (length(entry_href) BETWEEN 1 AND 500),
  version     text NOT NULL DEFAULT '1.2' CHECK (length(version) <= 20),
  mastery     smallint CHECK (mastery IS NULL OR mastery BETWEEN 0 AND 100),
  file_count  integer NOT NULL DEFAULT 0 CHECK (file_count >= 0),
  bytes       bigint NOT NULL DEFAULT 0 CHECK (bytes >= 0),
  grade_item_id uuid REFERENCES grade_items(id) ON DELETE SET NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

-- The package's own files. They live here rather than in object storage
-- because a package is small, self-contained and never edited: keeping it
-- beside the lesson means one place to back up and one to delete.
-- ponytail: fine to tens of megabytes; move to the object store if packages
-- start arriving with video inside them.
CREATE TABLE scorm_files (
  package_id   uuid NOT NULL REFERENCES scorm_packages(id) ON DELETE CASCADE,
  path         text NOT NULL CHECK (length(path) BETWEEN 1 AND 500),
  content_type text NOT NULL DEFAULT 'application/octet-stream' CHECK (length(content_type) <= 120),
  body         bytea NOT NULL,
  PRIMARY KEY (package_id, path)
);

-- What one learner has reported back. The names are SCORM's own, kept as they
-- come so nothing is lost in translation.
CREATE TABLE scorm_states (
  package_id    uuid NOT NULL REFERENCES scorm_packages(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  cmi           jsonb NOT NULL DEFAULT '{}'::jsonb,
  lesson_status text NOT NULL DEFAULT 'not attempted' CHECK (length(lesson_status) <= 40),
  score_raw     numeric(6,2),
  suspend_data  text NOT NULL DEFAULT '' CHECK (length(suspend_data) <= 64000),
  location      text NOT NULL DEFAULT '' CHECK (length(location) <= 1000),
  total_time_s  integer NOT NULL DEFAULT 0 CHECK (total_time_s >= 0),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (package_id, user_id)
);

CREATE INDEX scorm_files_package_idx ON scorm_files (package_id);

CREATE TRIGGER scorm_states_touch BEFORE UPDATE ON scorm_states
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE scorm_packages ENABLE ROW LEVEL SECURITY;
ALTER TABLE scorm_packages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON scorm_packages
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE scorm_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE scorm_states FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON scorm_states
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

-- The files carry no tenant of their own; they are reached only through a
-- package, which is scoped.
ALTER TABLE scorm_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE scorm_files FORCE ROW LEVEL SECURITY;
CREATE POLICY through_the_package ON scorm_files
  USING (EXISTS (SELECT 1 FROM scorm_packages p
                 WHERE p.id = scorm_files.package_id AND p.tenant_id = current_tenant_id()))
  WITH CHECK (EXISTS (SELECT 1 FROM scorm_packages p
                      WHERE p.id = scorm_files.package_id AND p.tenant_id = current_tenant_id()));

GRANT SELECT, INSERT, UPDATE, DELETE ON scorm_packages TO fajr_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON scorm_files TO fajr_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON scorm_states TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scorm_states;
DROP TABLE IF EXISTS scorm_files;
DROP TABLE IF EXISTS scorm_packages;
-- +goose StatementEnd
