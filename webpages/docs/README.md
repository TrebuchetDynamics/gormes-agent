# Gormes docs site

This directory is the source for the Astro/Starlight documentation site. Use
`webpages/docs/` directly; the repository root no longer keeps a `docs` symlink
alias.

## Tracked structure

- `content/` — canonical Markdown source for the docs site and progress-backed
  public pages.
- `static/` — tracked static assets copied by Astro as-is.
- `src/` — Starlight config/style source; `src/content/` is generated from
  `content/` before dev/build.
- `scripts/` — Node helpers for content sync, progress artifacts, and local
  cleanup.
- `www-tests/` — Playwright smoke tests for the published docs surface.
- `*_test.go` — Go checks that keep docs content aligned with the CLI,
  progress data, install guides, and public claims.
- Root historical Markdown files — tracked compatibility/planning artifacts
  still referenced by docs tests or older links.
- `superpowers/specs/*` compatibility files are real files, not symlinks; the
  organized subfolders remain the source of responsibility grouping.

## Local generated/cache paths

These paths are intentionally ignored and can be removed when you are not
actively running the docs site:

- `node_modules/`
- `www-tests/node_modules/`
- `.astro/`
- `dist/`
- `public/`
- `src/content/`
- `www-tests/test-results/`
- `.hugo_build.lock`

Use:

```bash
npm run clean       # remove generated build/sync artifacts
npm run clean:deps  # also remove docs and Playwright node_modules
```

Recreate dependencies later with:

```bash
npm ci
(cd www-tests && npm ci)
```

## Common commands

```bash
npm run sync:content
npm run dev
npm run build
go test ./webpages/docs -count=1
```
