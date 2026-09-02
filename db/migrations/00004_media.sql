-- +goose Up
-- +goose StatementBegin

CREATE TYPE media_state AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE media_assets (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider     text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_]{1,30}$'),
  -- The provider's own handle: an embed URL, an object key, a transcode job id.
  external_ref text NOT NULL CHECK (length(external_ref) BETWEEN 1 AND 2048),
  state        media_state NOT NULL DEFAULT 'pending',
  kind         lesson_kind NOT NULL DEFAULT 'video',
  title        text NOT NULL DEFAULT '' CHECK (length(title) <= 300),
  duration_s   integer NOT NULL DEFAULT 0 CHECK (duration_s >= 0),
  byte_size    bigint NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
  content_type text NOT NULL DEFAULT '' CHECK (length(content_type) <= 255),
  error        text NOT NULL DEFAULT '' CHECK (length(error) <= 2000),
  metadata     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE lessons ADD COLUMN media_id uuid REFERENCES media_assets(id) ON DELETE SET NULL;

CREATE INDEX media_tenant_state_idx ON media_assets (tenant_id, state);
CREATE INDEX lessons_media_idx ON lessons (media_id) WHERE media_id IS NOT NULL;

CREATE TRIGGER media_touch BEFORE UPDATE ON media_assets FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE media_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE media_assets FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON media_assets
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

-- Bandwidth is metered from the first migration, because you cannot price what
-- you did not measure.
CREATE TABLE media_delivery (
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  day        date NOT NULL,
  requests   bigint NOT NULL DEFAULT 0 CHECK (requests >= 0),
  bytes      bigint NOT NULL DEFAULT 0 CHECK (bytes >= 0),
  PRIMARY KEY (tenant_id, day)
);

ALTER TABLE media_delivery ENABLE ROW LEVEL SECURITY;
ALTER TABLE media_delivery FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON media_delivery
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON media_assets, media_delivery TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS media_delivery;
ALTER TABLE lessons DROP COLUMN IF EXISTS media_id;
DROP TABLE IF EXISTS media_assets;
DROP TYPE IF EXISTS media_state;
-- +goose StatementEnd
