-- +goose Up
-- +goose StatementBegin

CREATE TYPE delivery_state AS ENUM ('queued', 'sent', 'failed', 'skipped');

CREATE TABLE notifications (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind       text NOT NULL CHECK (kind ~ '^[a-z][a-z0-9_.]{1,60}$'),
  title      text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  body       text NOT NULL DEFAULT '' CHECK (length(body) <= 2000),
  data       jsonb NOT NULL DEFAULT '{}'::jsonb,
  read_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notification_deliveries (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  channel         text NOT NULL CHECK (channel ~ '^[a-z][a-z0-9_]{1,30}$'),
  destination     text NOT NULL CHECK (length(destination) BETWEEN 3 AND 320),
  body            text NOT NULL CHECK (length(body) BETWEEN 1 AND 2000),
  state           delivery_state NOT NULL DEFAULT 'queued',
  attempts        smallint NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  error           text NOT NULL DEFAULT '' CHECK (length(error) <= 1000),
  -- When this delivery may next be tried; backoff pushes it forward.
  run_after       timestamptz NOT NULL DEFAULT now(),
  sent_at         timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notifications_inbox_idx ON notifications (tenant_id, user_id, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;
CREATE INDEX deliveries_pending_idx ON notification_deliveries (run_after) WHERE state = 'queued';

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notifications
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE notification_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_deliveries
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON notifications, notification_deliveries TO fajr_app;

-- The dispatcher works across every tenant, so it runs through two audited
-- functions rather than being handed a scope it cannot have.
CREATE FUNCTION claim_deliveries(p_limit integer)
RETURNS SETOF notification_deliveries
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  UPDATE notification_deliveries SET attempts = attempts + 1, run_after = now() + interval '5 minutes'
  WHERE id IN (
    SELECT id FROM notification_deliveries
    WHERE state = 'queued' AND run_after <= now()
    ORDER BY run_after
    FOR UPDATE SKIP LOCKED
    LIMIT p_limit
  )
  RETURNING *
$$;

CREATE FUNCTION settle_delivery(p_id uuid, p_state delivery_state, p_error text, p_backoff interval)
RETURNS void
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  UPDATE notification_deliveries SET
    state = p_state,
    error = left(coalesce(p_error, ''), 1000),
    sent_at = CASE WHEN p_state = 'sent' THEN now() ELSE sent_at END,
    run_after = CASE WHEN p_state = 'queued' THEN now() + p_backoff ELSE run_after END
  WHERE id = p_id
$$;

REVOKE ALL ON FUNCTION claim_deliveries(integer), settle_delivery(uuid, delivery_state, text, interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION claim_deliveries(integer), settle_delivery(uuid, delivery_state, text, interval) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS settle_delivery(uuid, delivery_state, text, interval);
DROP FUNCTION IF EXISTS claim_deliveries(integer);
DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS delivery_state;
-- +goose StatementEnd
