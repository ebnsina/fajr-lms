-- +goose Up
-- +goose StatementBegin

ALTER TABLE class_sessions
  ADD COLUMN provider  text NOT NULL DEFAULT 'manual' CHECK (provider ~ '^[a-z][a-z0-9_]{1,30}$'),
  ADD COLUMN join_url  text NOT NULL DEFAULT '' CHECK (length(join_url) <= 2048),
  ADD COLUMN host_url  text NOT NULL DEFAULT '' CHECK (length(host_url) <= 2048),
  ADD COLUMN recording_media_id uuid REFERENCES media_assets(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE class_sessions
  DROP COLUMN IF EXISTS recording_media_id,
  DROP COLUMN IF EXISTS host_url,
  DROP COLUMN IF EXISTS join_url,
  DROP COLUMN IF EXISTS provider;
-- +goose StatementEnd
