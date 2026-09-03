-- +goose Up
-- +goose StatementBegin

-- Some schools want a leaderboard; many do not. Off unless a school says so.
ALTER TABLE tenants ADD COLUMN points_on boolean NOT NULL DEFAULT false;

-- One award, tied to the thing that earned it. The unique key on (kind, ref)
-- is what stops a lesson finished twice, or a webhook replayed, paying twice.
CREATE TABLE point_awards (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind       text NOT NULL CHECK (kind ~ '^[a-z][a-z_]{1,40}$'),
  ref_id     uuid NOT NULL,
  points     integer NOT NULL CHECK (points > 0 AND points <= 10000),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, user_id, kind, ref_id)
);

CREATE INDEX point_awards_board_idx ON point_awards (tenant_id, created_at DESC);
CREATE INDEX point_awards_user_idx ON point_awards (tenant_id, user_id);

-- A badge a school defines, and who has earned it.
CREATE TABLE badges (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name        text NOT NULL COLLATE human_name CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  description text NOT NULL DEFAULT '' CHECK (length(description) <= 300),
  emoji       text NOT NULL DEFAULT '' CHECK (length(emoji) <= 8),
  threshold   integer NOT NULL CHECK (threshold > 0),
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name)
);

CREATE TABLE badge_awards (
  badge_id   uuid NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  awarded_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (badge_id, user_id)
);

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['point_awards', 'badges', 'badge_awards'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS badge_awards;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS point_awards;
ALTER TABLE tenants DROP COLUMN IF EXISTS points_on;
-- +goose StatementEnd
