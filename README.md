# Fajr LMS

A learning platform for schools, madrasahs, colleges and teachers in South Asia
and the Gulf. It teaches, grades, collects the fees and keeps the guardians
informed — in Arabic, Bengali or English, on the phones learners already own.

API first: a Go service holds every rule, and the SvelteKit front end consumes
it like any other client would. A mobile app can come later without a rewrite.

## What it does today

- **Courses** — sections and lessons of any kind, reordered as the term changes,
  published one at a time. Video and audio arrive as pasted links; media is a
  plug, so a transcoder of your own drops in later.
- **Assessment** — quizzes with automatic and hand grading, assignments with
  file hand-in and a late policy the server decides, a grading queue, and a
  weighted gradebook.
- **Attendance** — take the register in one pass; an absence reaches the learner
  and anyone listed as their guardian the same morning.
- **Money** — bKash, SSLCommerz, and a bank slip somebody on staff approves.
  Approving enrolls the learner on the spot.
- **Certificates** — awarded on a finished course, with a serial anyone can
  check on a public page.
- **A website** — every school gets public pages built from validated
  plain-text sections, with eight templates written for both regions.
- **Many schools, one sign-in** — every school is isolated by Postgres row
  level security, not by application code that could forget.

`docs/build-plan.html` is the roadmap, the competitor research and the reasoning
behind each decision. Institution management (SIS/ERP) is deliberately deferred;
the plan names the conditions that should start it.

## Layout

| Path | What it is |
| --- | --- |
| `cmd/api` | The API server |
| `cmd/migrate` | Goose migrations, run as the database owner |
| `internal/api` | HTTP handlers, one file per area |
| `internal/site` | The page-builder blocks and their validation |
| `internal/media`, `internal/payment`, `internal/notify` | Provider seams |
| `db/migrations`, `db/queries` | Schema, and the SQL sqlc generates from |
| `web` | The SvelteKit front end — see `web/README.md` |
| `docs/build-plan.html` | The plan |

## Running it

Postgres 17 and Go 1.27. Copy `.env.example` to `.env` and fill it in.

```sh
ADMIN_DATABASE_URL=... make migrate   # migrations run as the owner, not the API role
make sqlc                             # regenerate after changing db/queries
go run ./cmd/api                      # the API
cd web && npm install && npm run dev  # the front end
```

The API connects as `fajr_app`, a role with `NOBYPASSRLS`. Every tenant-scoped
query runs inside a transaction that sets `app.tenant_id`; anything that has to
run before a tenant exists goes through one named `SECURITY DEFINER` function.

## Tests

```sh
FAJR_DATABASE_URL=... go test ./...   # the API tests need a database
cd web && npm run check && npm run build
```

CI runs both on every push.
