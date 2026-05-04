# Gormes Interface Checklist

Use this before accepting an interface design.

## Callers

- Which CLI/API/channel/tool/memory/provider code calls it?
- Is the public contract stable enough for tests?
- Can callers use it without knowing provider/storage internals?

## Depth

- Does the interface hide meaningful complexity?
- Would deleting it spread complexity across callers?
- Are there real adapters, or only hypothetical future adapters?

## Compatibility

- Which Hermes or Honcho behavior must be preserved?
- Are public names compatible where users/tools depend on them?
- Is internal naming still Go-native and Gormes-owned?

## Testing

- What public behavior proves the interface?
- Can tests run without live credentials or network services?
- Which fixtures establish compatibility?

## Failure Modes

- How does degraded mode appear in doctor/status/logs?
- Which errors are typed or classified?
- What should callers do on partial failure?
