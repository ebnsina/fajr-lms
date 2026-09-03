-- +goose Up
-- +goose StatementBegin

-- A school lays out its own certificate: where the name sits, how large, and
-- what paper it is printed on. One layout per school; without one, the
-- certificate keeps the design we ship.
CREATE TABLE certificate_layouts (
  tenant_id       uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  fields          jsonb NOT NULL DEFAULT '[]'::jsonb,
  background      bytea,
  background_type text NOT NULL DEFAULT '' CHECK (length(background_type) <= 60),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER certificate_layouts_touch BEFORE UPDATE ON certificate_layouts
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE certificate_layouts ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_layouts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON certificate_layouts
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON certificate_layouts TO fajr_app;

-- Verifying a certificate happens with nobody signed in, so the layout it is
-- drawn with is reached through a named function like the rest of that path.
CREATE FUNCTION certificate_layout_for(p_tenant_id uuid) RETURNS SETOF certificate_layouts
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM certificate_layouts WHERE tenant_id = p_tenant_id
$$;

-- Which school issued a serial, so the public page can find its layout.
CREATE FUNCTION certificate_tenant(p_serial text) RETURNS SETOF uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT tenant_id FROM certificates WHERE serial = p_serial
$$;

REVOKE ALL ON FUNCTION certificate_layout_for(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION certificate_tenant(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION certificate_layout_for(uuid) TO fajr_app;
GRANT EXECUTE ON FUNCTION certificate_tenant(text) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS certificate_tenant(text);
DROP FUNCTION IF EXISTS certificate_layout_for(uuid);
DROP TABLE IF EXISTS certificate_layouts;
-- +goose StatementEnd
