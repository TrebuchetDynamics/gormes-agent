# Gormes.ai

Astro + Tailwind landing page for current Gormes trunk.

The site should reflect the shipped moat layers truthfully: the zero-CGO Go
shell, the Go-native tool registry, Telegram/Discord on the shared gateway,
Route-B resilience, and the progress-driven Phase-2 shipping boundary. It
should not regress into a Phase-1-only story or hardcode stale proof claims.

Astro owns the public homepage at `/`, static assets at `/static/*`, and
installer aliases at `/install.sh`, `/install.ps1`, and `/install.cmd`.
Tailwind is wired through the Tailwind v4 Vite plugin in `astro.config.mjs`.
The former Go-rendered site is deprecated and preserved under
`legacy/go-renderer/` for reference only.

## Layout

- `src/pages/index.astro` - homepage route, structure, metadata, and inline
  copy-button behavior.
- `src/data/landing.js` - landing-page copy, benchmark/progress helpers, and
  roadmap derivation.
- `src/data/progress.json` - generated mirror of
  `../docs/content/building-gormes/architecture_plan/progress.json`.
- `src/data/benchmarks.json` - generated mirror of `../../benchmarks.json`.
- `src/styles/global.css` - Tailwind import, theme fonts, and base focus/copy
  states.
- `public/static/*` - favicon, social card, and landing visual assets.
- `public/install.*` - generated installer aliases served as static files.
- `scripts/sync-assets.mjs` - copies canonical installers, progress, benchmark,
  and static assets before dev/build.
- `tests/home.spec.mjs` - Playwright smoke test for the homepage.
- `legacy/go-renderer/` - deprecated Go renderer retained for rollback only.

## Installer Surface

The site serves three installer assets, one per supported user shell:

| Path | Source | Audience |
|------|--------|----------|
| `/install.sh` | `../../install.sh` | Linux, macOS, Termux, WSL |
| `/install.ps1` | `../../scripts/install.ps1` | Windows PowerShell 5.1+ / pwsh 7+ |
| `/install.cmd` | `../../scripts/install.cmd` | CMD wrapper that launches the PowerShell installer |

The Unix installer is source-backed like Hermes Agent: it clones or updates a
managed checkout, builds `gormes`, publishes a stable global command, verifies
the binary, starts `gormes setup` when a terminal is available, and reruns as an
update flow. Use `--skip-setup` or `GORMES_SKIP_SETUP=1` to defer that wizard.
Termux publishes to `$PREFIX/bin`. Root Linux publishes to `/usr/local/bin`;
non-root installs publish to a user-scoped bin directory unless overridden.

The landing page should keep the inspect-first installer and source build paths
visible until signed binaries, Homebrew, and Scoop/Winget manifests land.
The `/install.*` URLs remain convenience aliases, not the primary trust story.

## Local Development

```bash
cd webpages/landing
npm install
npm run dev
```

Build the static site:

```bash
npm run build
```

Preview the production build:

```bash
npm run preview
```

## Verification

Install the browser-test dependency once per checkout:

```bash
npm install
```

Run the browser smoke test:

```bash
npm run test:e2e
```

The Playwright config launches the Astro dev server, so no separate app process
is needed for the smoke test.

## Content Updates

- Edit `src/data/landing.js` to change copy, CTAs, or roadmap framing.
- Edit `src/pages/index.astro` to change structure.
- Edit `src/styles/global.css` for Tailwind theme/base states; prefer Tailwind
  utility classes in `.astro` components for page layout.
- Run `npm run build` to sync installer/progress/benchmark mirrors and compile
  the static site. There is no `webpages/landing/content/` Markdown tree
  in this module; the homepage roadmap comes from the mirrored JSON data.

The page intentionally avoids client-side JavaScript except for the bounded
copy-button behavior. The homepage should remain readable and useful with
scripts disabled.
