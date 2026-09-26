@AGENTS.md

# 红线 (red-thread)

A couple's photo album that doubles as a wedding invitation. Next.js 16 (App Router, Turbopack, `src/proxy.ts`
instead of middleware), React 19, hand-written CSS in `src/styles/` (no UI framework). See README.md (Chinese).

## Commands

- `npm run dev` — zero config: PGlite in `.data/pglite`, photos in `.data/uploads`, dev login `us@love.local` / `together`
- `npm run typecheck`, `npm run lint`, `npm test` (node:test, `src/**/*.test.ts`), `npm run build`
- `node scripts/demo-art.mjs` regenerates `public/demo/*.svg` (sample album illustrations)

## Conventions

- Pages that read the database export `dynamic = 'force-dynamic'` (Cache Components are not enabled).
- Every page under `/us` and `/admin` calls `requireSession()`; every server action / route handler calls
  `assertSession()` / `getSession()` itself. The proxy is only an optimistic redirect.
- The browser only ever receives `PhotoCard` (`toCards`), never raw `Photo` rows: GPS, camera and the original
  file key stay on the server. Pass `{ forUs: true }` only on signed-in pages.
- Guests see public photos only (`visibility = 'public'`), and public moments only. Private photos inside a
  public moment are shown as a count (`sealed`), nothing else.
- Dates are stored as text (`YYYY-MM-DD`, wall-clock `YYYY-MM-DDTHH:mm:ss` for `taken_at`, ISO UTC for
  `created_at`) so PGlite and node-postgres return identical values. Calendar maths lives in `src/lib/dates.ts`.
- Schema changes: append idempotent statements to `src/lib/schema.ts` (`ADD COLUMN IF NOT EXISTS`).
- Storage keys: `photos/<id12>-<token20>-{lg,sm,orig}.<ext>`, `media/<token20>.<ext>`; validated by
  `src/lib/ids.ts`. Images are resized and stripped of EXIF in the browser (`src/lib/image-client.ts`).
- Read `NEXT_PUBLIC_*` values that the server needs at runtime through `src/lib/env.ts`
  (`process.env.NEXT_PUBLIC_X` is inlined at build time).
- Forms using `useActionState` must return what the user typed on error: React 19 resets the form after an action.
