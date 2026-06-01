# Taxonomy And Owned Divergence

Use this when parity labels, feature-map headings, row names, public progress
wording, or package terminology no longer match the upstream contract.

## Parity Definitions

| Label | Meaning |
|---|---|
| `strict` | Gormes must match upstream names, inputs, outputs, errors, side effects, and registration exactly. |
| `functional` | Gormes preserves the user/operator contract, but Go internals or provider shape may differ. This is the default target. |
| `owned` | Gormes intentionally diverges or extends Hermes. The row must explain why and how compatibility is preserved or why it is not required. |
| `excluded` | Upstream behavior is intentionally not part of Gormes. Record source-backed rationale and user-visible risk. |

When strict and functional parity conflict, prefer functional parity only when
the difference is documented as `owned` or the public Hermes contract remains
preserved.

## Owned Divergence Rule

Do not silently accept owned divergence. Make it durable when it is:

- hard to reverse;
- surprising without context;
- the result of a real trade-off;
- visible to users, operators, tools, docs, or future progress rows.

Record durable divergence in the progress row and, when repo docs have an ADR
lane for the affected area, propose or add an ADR-style decision note. If the
divergence is narrow and obvious, a progress-row compatibility note is enough.

## Safe Restructure Loop

1. Write `old name -> new name`, scope, source refs, and compatibility decision.
2. Use `rg -n` across `cmd`, `internal`, `docs`, `webpages`, skills, tests, and
   generated data before editing.
3. Preserve `progress.json` schema and row identity unless the split/merge is
   the point.
4. Keep public aliases unless a source-backed decision says the break is
   intentional.
5. Regenerate derived docs with `go run ./cmd/progress write` when progress or
   generated surfaces changed.
6. Re-run `rg -n` for old terms. Remaining refs must be intentional history,
   migration, or compatibility notes.
7. Validate the touched surfaces.

Large restructures should land as no-runtime-behavior taxonomy migrations, then
separate builder rows for implementation.
