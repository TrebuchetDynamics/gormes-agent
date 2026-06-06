# Gormes.ai Hard Cutover Design Spec

**Date:** 2026-04-19
**Author:** Xel (via Codex brainstorm)
**Status:** Approved design direction; ready for planning
**Scope:** Hard-cut the public website module and its validating docs/tests from `www.gormes.io` to `www.gormes.ai`, while keeping Node/Playwright isolated to the website module and tightening the site-local `README.md`.

---

## 1. Purpose

The project now owns `gormes.ai`, and the public website should stop carrying an `.io` identity split between:

- public copy;
- filesystem layout;
- Go module path;
- docs tests;
- developer instructions.

This work is about removing that identity debt cleanly.

When it is done, the website module, its local docs, and its tests should all agree that the public home is `www.gormes.ai`.

---

## 2. Locked Decisions

### 2.1 Hard cutover

This is a hard cutover to `.ai`, not a branding-only pass.

The implementation must update:

- public site branding;
- website folder name;
- Go module path;
- local imports inside the site module;
- docs tests that validate the landing-page implementation;
- website-local README instructions.

It must not leave the landing-page module in a split state where the public domain says `.ai` but the code and docs still present `www.gormes.io` as the canonical name.

### 2.2 Canonical identity

The canonical public identity for the site is:

- `gormes.ai` as the project domain;
- `www.gormes.ai` as the website host used in code/docs/copy where a concrete hostname is needed.

If `.io` is mentioned at all, it should only appear as a compatibility or redirect note in future operational docs, not as co-equal branding.

### 2.3 README requirement

The website module must have its own `README.md` that explains:

- what the module is;
- how to build it;
- how to run it locally;
- how to run Go tests;
- how to run the Playwright smoke test;
- where templates, static assets, and content live.

This README is a developer note for the website module directory, not a marketing page.

### 2.4 Dependency boundary

Node.js is acceptable only as website-module test tooling.

Allowed:

- `www.gormes.ai/package.json`;
- `@playwright/test`;
- lockfile and Playwright config inside the website module.

Not allowed:

- introducing Node dependencies into `gormes/`;
- making the Go agent depend on Node for build, run, or test;
- turning the public site into a JavaScript-rendered frontend.

The page itself remains Go-rendered and script-free.

### 2.5 Redirect stance

This task does not require implementing runtime redirects from `.io` to `.ai`.

However, the code and docs should avoid baking in assumptions that make a future `.io` secondary-domain redirect awkward. The canonical naming should be `.ai`; redirect mechanics can be handled later at the edge or by simple host-aware middleware if needed.

---

## 3. Scope

### 3.1 In scope

- Rename `www.gormes.io/` to `www.gormes.ai/`.
- Update the website Go module path to match the renamed directory.
- Update imports and log lines inside the website module.
- Update landing-page tests and docs tests that point at the website module path.
- Update site-facing strings such as page title or README branding where they still say `.io`.
- Tighten the website README so it clearly documents build/run/test workflow.

### 3.2 Out of scope

- Rewriting unrelated historical docs that mention `gormes.io` as part of older planning context.
- DNS, registrar, CDN, or reverse-proxy configuration.
- Implementing host-based redirect behavior.
- Changing the core landing-page narrative beyond what is needed for the domain cutover.
- Moving Node tooling outside the website module.

Older documents may still mention `.io` historically. That is acceptable unless those docs are part of the active landing-page validation surface.

---

## 4. Design

### 4.1 Filesystem identity

The website module directory should become:

`www.gormes.ai/`

That directory remains a standalone Go module and continues to own:

- the server entrypoint;
- `internal/site` templates/content/assets;
- Playwright smoke tests;
- its own README;
- its own package manifest for browser testing.

### 4.2 Module identity

The Go module path should be renamed to match the new directory identity.

All internal imports within the website module must be updated accordingly. There should be no remaining import path that references `www.gormes.io`.

### 4.3 Public copy

The site should present itself as `Gormes.ai`, not `Gormes.io`.

This includes, where present:

- page title;
- README title;
- logging strings;
- any developer-facing labels that name the website module directly.

The approved landing-page product framing does not need to change otherwise. The hero, roadmap, and audience positioning remain valid.

### 4.4 README contract

The website README should cover these sections at minimum:

1. What `www.gormes.ai` is.
2. Directory layout.
3. Local run command.
4. Build command.
5. Go test command.
6. Playwright smoke test command.
7. Notes on embedded assets and script-free rendering.

It should be practical, short, and specific to the current implementation.

### 4.5 The indie personality line

If the landing-page copy or README includes a small indie-human note such as the "space lobster soul" flavor, that is acceptable so long as it stays secondary to the technical trust story.

This project can have personality, but the rename task should not expand the marketing surface just to inject voice.

---

## 5. Testing Strategy

This rename is a behavior change and must follow TDD discipline.

### 5.1 Tests to update first

Before implementation code is changed, the relevant tests should be updated to expect the `.ai` paths/strings.

This includes:

- website Go tests under `www.gormes.ai/internal/site`;
- docs tests under `gormes/docs`;
- Playwright smoke tests and any path-sensitive config they depend on.

### 5.2 Verification commands

After the rename, the verification sequence must be:

- `cd www.gormes.ai && go test ./...`
- `cd gormes && go test ./docs`
- `cd www.gormes.ai && make build`
- `cd www.gormes.ai && npm run test:e2e`

If `npm install` is needed in a fresh checkout, the README should say so explicitly.

---

## 6. Acceptance Criteria

The task is complete when all of the following are true:

1. The website module directory is named `www.gormes.ai`.
2. The website Go module path and imports no longer reference `www.gormes.io`.
3. The website README exists at `www.gormes.ai/README.md` and documents build/run/test clearly.
4. The landing-page validation docs/tests that currently track the website module path now reference `www.gormes.ai`.
5. The page remains Go-rendered and does not gain a runtime JavaScript dependency.
6. Node/Playwright tooling remains isolated to the website module.
7. The verification commands in Section 5.2 pass on the renamed module.

---

## 7. Risks and Controls

### 7.1 Rename fallout

Directory renames can leave stale path references in:

- Go imports;
- docs tests;
- README instructions;
- Playwright config;
- shell commands in docs.

Control:

- update tests first;
- use targeted search for both `gormes.io` and `www.gormes.io`;
- verify from the renamed directory, not the old one.

### 7.2 Scope creep

The rename could sprawl into a full-site rewrite or a sweeping docs cleanup.

Control:

- only update active landing-page surfaces and the website module;
- leave historical docs alone unless they are part of current validation.

### 7.3 Dependency creep

The presence of Playwright can tempt broader Node usage.

Control:

- keep Node assets scoped to `www.gormes.ai/`;
- keep the page itself Go-rendered;
- do not introduce frontend build tooling for this task.

---

## 8. Summary

This is not just a copy edit. It is an identity alignment pass.

`www.gormes.ai` should be the name users see, the folder developers work in, the module path Go builds, the path docs tests validate, and the README developers follow.

The website keeps its current product story. The hard cutover simply makes the implementation and the brand agree on where the project lives.
