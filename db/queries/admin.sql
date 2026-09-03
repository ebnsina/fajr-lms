-- name: StaffByEmail :one
SELECT * FROM staff_by_email(@email);

-- name: StaffRole :one
SELECT * FROM staff_role(@user_id);

-- name: TouchStaff :exec
SELECT staff_seen(@user_id);

-- name: LogStaffAction :exec
SELECT staff_log(@user_id, @action, @subject, @detail);

-- name: AdminLeads :many
SELECT * FROM admin_leads(@state, @query, @limit_to, @offset_by);

-- name: AdminSetLead :one
SELECT * FROM admin_set_lead(@id, @state, @note);

-- name: AdminTenants :one
SELECT * FROM admin_tenants(@query, @limit_to, @offset_by);

-- name: AdminTenant :one
SELECT * FROM admin_tenant(@tenant_id);

-- name: AdminOverview :one
SELECT * FROM admin_overview();
