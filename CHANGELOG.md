# Changelog

All user-facing changes. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Multi-tenant data model: tenants (institution, creator, corporate), global
  users, and per-tenant memberships with roles.
- Tenant isolation enforced by Postgres row-level security, not application
  code, with a test suite that proves cross-tenant reads and writes fail.
- Bidi-ready schema: per-tenant text direction and locale, ICU collation so
  Latin, Arabic and Bengali names sort correctly, trigram index for name search.
- `GET /readyz` reports database reachability.
- HTTP API skeleton with `GET /healthz`, JSON error responses, request ids on
  every response, and graceful shutdown.
