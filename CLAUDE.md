# Fajr LMS

Go API (API-first) + SvelteKit web + Expo mobile later. See `docs/build-plan.html`.

## Git

- Author every commit as `ebnsina <ebnsina.me@gmail.com>`. Set per-repo:
  ```
  git config user.name "ebnsina"
  git config user.email "ebnsina.me@gmail.com"
  ```
- **No `Co-Authored-By` trailer.** No other identity on commits, ever.
- Remote: `git@github-es:ebnsina/fajr-lms.git` (github-es SSH host alias).

## Changelog

`CHANGELOG.md` is updated with every user-facing change, in the same commit.
Internal refactors and chores do not get an entry.

## Docs and research

Verify against current docs before writing code against any library or API —
use `ctx7` / Context7. Never write API calls from memory.

## Code quality

- Modular, clean, maintainable. Small files, clear boundaries.
- **Comments: 1 line, 2 max. Hard rule, no exceptions.**
- Handle every error path explicitly: 404, 500, network failure, validation,
  edge cases. No unhandled branches, no silent `_`.
- Formatting (dates, numbers, currency, relative time, lists, plurals) uses the
  `Intl` Web API on the frontend. No hand-rolled formatters.
  Go side uses `golang.org/x/text` for the equivalent — same rule, no custom logic.
