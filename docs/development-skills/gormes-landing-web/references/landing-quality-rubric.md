# Landing Quality Rubric

Use this reference when `gormes-landing-web` needs a broad content audit,
homepage rewrite, or UI polish pass.

## First-Viewport Test

A stranger should understand within 10 seconds:

1. Gormes runs AI agents as a Go-native binary.
2. It solves install drift, runtime fragility, and dropped-stream failures.
3. It is early-stage and not production-stable yet.
4. It does not require a Hermes process.
5. There is a clear install or try-now action.

## Content Checks

Rate each as `clear`, `partial`, or `missing`:

1. **Headline** - outcome first, not internal architecture first.
2. **Status** - current limits are visible and not buried.
3. **Try Path** - install, offline smoke test, doctor, and provider-backed run are obvious.
4. **Proof** - binary size, tests, generated progress, deploy workflows, and docs links are current.
5. **No Stale Dependency Claims** - no copy implies Hermes must be running.
6. **Audience Fit** - speaks to operators and builders with concrete pains, not vague personas.
7. **Feature Translation** - jargon becomes outcomes: streams recover, installs stop drifting, memory stays local.
8. **Depth Links** - architecture and Goncho details link out instead of overloading the homepage.

## UI Checks

1. First viewport has a clear brand/product signal, primary action, and visible next-section hint.
2. Text does not overlap or overflow on desktop, tablet, or mobile.
3. Controls and CTAs are visually distinct and keyboard-accessible.
4. Cards are used for repeated items only; avoid nested cards.
5. Color and spacing support scanning; avoid one-note palettes and decorative clutter.
6. Installer snippets are copyable and fit their containers.
7. The page remains useful with scripts disabled.

## Validation Bias

Prefer automated proof:

- Go render/server tests for content and static export.
- Playwright for first viewport, responsive layout, CTAs, and stale-copy regressions.
- Screenshot inspection when CSS or layout changes are material.
