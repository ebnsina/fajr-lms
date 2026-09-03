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

## Frontend

- **No third-party component libraries.** Everything is built in the repo.
  Exceptions, both animation-only: GSAP (with ScrollTrigger) for motion on the
  marketing pages, and Lucide for icons, imported one icon at a time.
- Design tokens live in `web/src/app.css`. Brand is emerald; radii, control
  height and focus treatment are tokens — never one-off values.
- The product is set in Cabin with Geist Mono for numbers and code. The
  marketing pages have their own voice: Clash Display for headings, Excon for
  body. Every face is self-hosted in `web/static/fonts` and subset to Latin, so
  Arabic and Bengali fall through to Noto.
- Motion is skipped under `prefers-reduced-motion` **and** when the tab is
  hidden — a hidden tab has no animation frames, so an intro would leave the
  page parked at opacity zero.
- `web/.prettierrc` holds the format: tabs, single quotes, 100 columns. Run
  `npx prettier --write "src/**/*.ts"` from `web/`; `.svelte` files need the
  Svelte plugin, which is not installed, so leave them to hand formatting.

## Claims

Never describe a feature on a public page as available unless it exists in the
codebase. Work that is planned is written as planned, in as many words.

## Fajr AI

Anything that is Fajr AI is marked with the fluid orb (`$lib/components/FluidOrb.svelte`),
sized to the icons around it. Never a sparkles or wand icon.
