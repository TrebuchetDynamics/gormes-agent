# pkg/ — Public Facade

## Responsibility

`pkg/` is the stable external import surface for Gormes. It should re-export selected runtime contracts from `internal/` without owning implementation logic.

## Packages

| Path | Responsibility |
|---|---|
| `pkg/gormes` | Type aliases for the public Phase-1 runtime surface: LLM client/stream events, kernel render/platform events, and the runtime bridge seam. |

## Rules

- Keep behavior in `internal/`; do not add implementation logic under `pkg/`.
- Prefer type aliases over wrapper types when preserving compatibility across internal refactors.
- Treat additions here as public API: add tests or compile checks when exposing new contracts.
- Keep comments clear about which internal package owns the real type.

## Current Flow

```text
external import github.com/TrebuchetDynamics/gormes-agent/pkg/gormes
  -> aliases internal/llm, internal/kernel, internal/runtime/bridge contracts
  -> implementation remains in internal packages
```

## Validation

- Public facade compile check: `go test ./pkg/... -count=1`
- Full compatibility gate when changing aliases: `go test ./... -count=1`
