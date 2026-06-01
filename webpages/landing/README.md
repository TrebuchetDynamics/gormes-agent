# Gormes.ai

Astro + Tailwind landing page for current Gormes trunk.

The site should sell one clear idea first: run AI agents anywhere from one Go
binary. Supporting copy should stay truthful about the Go runtime,
Telegram/Discord/Slack gateways, experimental Navivox phone pairing, release
proof, and current roadmap boundaries without turning the homepage into a
feature dump.

Astro owns the public homepage at `/`, static assets at `/static/*`, and
Windows installer aliases at `/install.ps1` and `/install.cmd`.
Tailwind is wired through the Tailwind v4 Vite plugin in `astro.config.mjs`.
The former Go-rendered site is deprecated and preserved under
`legacy/go-renderer/` for reference only; active builds no longer sync assets or
progress data into it.

## Layout

- `src/pages/index.astro` - homepage route, structure, and inline copy-button
  behavior.
- `src/pages/built-with.astro` - real-deployments proof page.
- `src/components/BaseDocument.astro` - shared document metadata, icons,
  font preconnects, global CSS import, and Astro runtime marker.
- `src/data/landing.js` - landing-page copy, trust proof, benchmark/release
  labels, use cases, and CTA data.
- `src/data/benchmarks.json` - generated mirror of `../../benchmarks.json`.
- The homepage links to roadmap docs instead of rendering roadmap bullets; the
  former `src/data/progress.json` verbatim mirror is intentionally untracked.
- `src/styles/global.css` - Tailwind import, theme fonts, and base focus/copy
  states.
- `public/static/*` - favicon, social card, and landing visual assets.
- `public/install.sh`, `public/install.ps1`, and `public/install.cmd` - generated installer aliases served as static files.
- `scripts/sync-assets.mjs` - copies canonical installers, benchmark data, and
  release metadata before dev/build.
- `tests/home.spec.mjs` - Playwright smoke test for the homepage.
- `legacy/go-renderer/` - deprecated Go renderer retained for rollback/reference only; not part of the active Astro build or asset-sync path.

## Installer Surface

The site serves the short custom-domain installer assets used by public copy.
Unix users can run the Hermes-style command `curl -fsSL https://gormes.ai/install.sh | bash`; Windows users keep the PowerShell/CMD aliases.

| Path | Source | Audience |
|------|--------|----------|
| `/install.sh` | `../../install.sh` | Linux, macOS, WSL2, Termux |
| `/install.ps1` | `../../scripts/install.ps1` | Windows PowerShell 5.1+ / pwsh 7+ |
| `/install.cmd` | `../../scripts/install.cmd` | CMD wrapper that launches the PowerShell installer |

The Unix installer is source-backed like Hermes Agent: it clones or updates a
managed checkout, builds `gormes`, publishes a stable global command, verifies
the binary, starts `gormes setup` when a terminal is available, and reruns as an
update flow. Use `--skip-setup` or `GORMES_SKIP_SETUP=1` to defer that wizard.
Termux publishes to `$PREFIX/bin`. Root Linux publishes to `/usr/local/bin`;
non-root installs publish to a user-scoped bin directory unless overridden.

The landing page should lead with the `https://gormes.ai/install.sh` Unix
installer command and keep source build paths visible until package-manager
manifests land. `/install.ps1` and `/install.cmd` remain Windows convenience
aliases.

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
  the static site. The prebuild sync copies canonical benchmark data from the
  repo root; when `bin/gormes` exists it can refresh measurements first with
  `GORMES_WWW_REFRESH_BENCHMARKS=1`. There is no
  `webpages/landing/content/` Markdown tree in this module; the homepage
  roadmap comes from the generated progress data.

The page intentionally avoids client-side JavaScript except for the bounded
copy-button behavior. The homepage should remain readable and useful with
scripts disabled.
