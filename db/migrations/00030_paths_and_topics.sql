-- +goose Up
-- +goose StatementBegin

-- What a course is about, so a catalog of forty is navigable.
CREATE TABLE topics (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name       text NOT NULL COLLATE human_name CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  slug       text NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,60}$'),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug)
);

CREATE TABLE course_topics (
  course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  topic_id  uuid NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  PRIMARY KEY (course_id, topic_id)
);

-- A path is courses in an order, worked through one after another. A bundle is
-- courses bought together. They are the same shape, so they are one table with
-- a kind: keeping them apart would mean writing every join twice.
CREATE TYPE collection_kind AS ENUM ('path', 'bundle');

CREATE TABLE collections (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  kind         collection_kind NOT NULL,
  slug         text NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
  title        text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  summary      text NOT NULL DEFAULT '' CHECK (length(summary) <= 500),
  dir          text_dir NOT NULL DEFAULT 'auto',
  status       publish_status NOT NULL DEFAULT 'draft',
  -- A bundle has its own price; a path is worked through, not sold.
  price_minor  bigint NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  currency     char(3) NOT NULL DEFAULT 'BDT' CHECK (currency ~ '^[A-Z]{3}$'),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug),
  CONSTRAINT only_a_bundle_has_a_price CHECK (kind = 'bundle' OR price_minor = 0)
);

CREATE TABLE collection_courses (
  collection_id uuid NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  course_id     uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  position      double precision NOT NULL,
  PRIMARY KEY (collection_id, course_id)
);

CREATE INDEX collection_courses_order_idx ON collection_courses (collection_id, position);
CREATE INDEX course_topics_topic_idx ON course_topics (topic_id);

CREATE TRIGGER collections_touch BEFORE UPDATE ON collections FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['topics', 'course_topics', 'collections', 'collection_courses'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS collection_courses;
DROP TABLE IF EXISTS collections;
DROP TYPE IF EXISTS collection_kind;
DROP TABLE IF EXISTS course_topics;
DROP TABLE IF EXISTS topics;
-- +goose StatementEnd
