# Fajr LMS — web

The SvelteKit front end: the product itself, each school's public website, and
our own marketing pages. It talks to the Go API and holds no database of its
own.

## Running it

```sh
npm install
npm run dev          # http://localhost:5173
```

The API has to be running too, and its address comes from the environment:

```sh
FAJR_API_URL=http://localhost:8080   # default
```

## What is where

| Path | What it is |
| --- | --- |
| `src/routes/(app)` | The product: courses, grading, gradebook, attendance, payments, the website builder |
| `src/routes/(site)/site/[tenant]` | A school's public website, rendered from its published pages |
| `src/routes/(marketing)` | Our own landing, pricing and sign-up pages |
| `src/routes/login`, `/tenant` | Signing in and choosing which school you are working in |
| `src/lib/components` | Everything shared. No third-party component library |
| `src/lib/actions/motion.ts` | GSAP reveals for the marketing pages, skipped when motion is unwanted |
| `src/app.css` | Every design token: colour, radii, control height, type |

## Checks

```sh
npm run check        # svelte-check: types and template errors
npm run build        # production build, adapter-node
npx prettier --write "src/**/*.ts"
```

`.svelte` files are formatted by hand: the Prettier Svelte plugin is not
installed, and Prettier refuses the files without it.

## Conventions

- Tokens, never one-off values. Buttons, inputs and badges are `rounded-xl`,
  cards `rounded-3xl`, and every control shares one height.
- Logical properties (`inline-size`, `margin-inline`) rather than left and
  right, because half the schools read right to left.
- `dir="auto"` on anything a person typed, so an Arabic title sits correctly
  inside an English table.
- Motion is skipped under `prefers-reduced-motion` and in a hidden tab.
