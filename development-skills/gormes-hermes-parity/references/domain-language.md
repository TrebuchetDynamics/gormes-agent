# Domain Language

Use these terms exactly. They keep parity reports short and navigable.

| Term | Meaning |
|---|---|
| Active upstream contract | The current Hermes/Honcho source, tests, or docs that define the user-visible behavior for the selected surface. |
| Behavior atom | A single observable unit: surface, trigger, visible output, side effect, degraded behavior, upstream refs, Gormes refs, status, row, validation, risk. |
| Behavior-fidelity pair | One upstream behavior family plus the closest Gormes surface or row that should preserve it. |
| Source checkout | A read-only evidence repo such as `./hermes-agent`, `./honcho`, or their fallbacks. Not a runtime dependency. |
| Runtime home | The live Gormes state directory, usually `GORMES_HOME` or `~/.gormes`. Do not inspect private runtime homes unless the row requires sanitized migration/runtime fixtures. |
| Installer-managed checkout | The source clone or binary path used by a final-user install. Keep it distinct from the active development checkout. |
| Progress row | The canonical implementation unit in `webpages/docs/content/building-gormes/architecture_plan/progress.json`. |
| Owned divergence | An intentional Gormes behavior that differs from Hermes while preserving a documented compatibility boundary, or explicitly explaining why compatibility is not required. |
| Covered | Implemented, tested, and source-backed against the active upstream contract. |
| Planned | Represented by a builder-ready progress row with acceptance and tests. |
| Vague | A row exists but is too broad, ambiguous, missing tests, or missing source refs. |
| Missing | No useful Gormes code or progress row exists. |
| Stale-upstream | Existing refs target retired upstream behavior instead of the active contract. |
| Blocked | A required dependency, source checkout, credential, fixture, or interface decision is absent. |
| Excluded | Upstream behavior is intentionally not part of Gormes, with source-backed rationale and user-visible risk noted. |

When terminology is fuzzy, pause and rename the concept before row edits. If a
new term becomes load-bearing across skills, update this reference or the
appropriate repo-local skill instead of repeating prose in progress notes.
