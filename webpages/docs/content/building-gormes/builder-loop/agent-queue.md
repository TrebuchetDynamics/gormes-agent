---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Image generation managed-gateway provider binding

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Bind the existing image_generate runner/provider registry to the existing ManagedGatewayBridge with hermetic fake HTTP MCP gateway fixtures, so a configured managed image provider can generate the standard redacted image artifact envelope without live FAL/API credentials.
- Trust class: operator, system
- Ready when: ManagedGatewayBridge discovery, call passthrough, cancellation, auth-required, unavailable, schema-rejected, tool-call-failed, and circuit-breaker evidence are already fixture-covered., ImageGenRunner/ImageGenTool already support injected provider registries, plugin discovery refresh, configured-provider routing, artifact envelopes, and redacted provider errors., The builder can prove the binding with httptest and temp output directories; no live managed gateway, FAL key, model provider, or external network is required.
- Not ready when: The slice tries to port the entire Hermes managed gateway lifecycle/config UI instead of one image-generation provider binding., The implementation requires live NOUS/FAL/API credentials or real network gateway access to pass tests., The image artifact envelope bypasses the existing BuildImageGenerationEnvelope/redaction path or changes the public image_generate schema.
- Degraded mode: Auth-required, gateway-unavailable, schema-rejected, tool-call-failed, and circuit-breaker outcomes map to stable image-generation degraded evidence; bearer tokens, prompts, and raw gateway errors stay redacted.
- Fixture: `internal/tools/managed_tool_gateway_test.go fake HTTP MCP gateway plus internal/tools/imagegen/generation_test.go artifact-envelope fixtures`
- Write scope: `internal/tools/managed_tool_gateway.go`, `internal/tools/managed_tool_gateway_test.go`, `internal/tools/imagegen/`, `internal/tools/image_generation_provider.go`, `cmd/gormes/registry.go`, `cmd/gormes/registry_test.go`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/tools.json`
- Test commands: `go test ./internal/tools ./internal/tools/imagegen -run 'TestManagedGatewayBridge\|TestMCPCircuitBreakerManagedGatewayEvidence\|TestImageGen\|TestImageGeneration\|TestImageGenManagedGateway' -count=1`, `go test ./cmd/gormes -run 'TestBuildDefaultRegistry.*ImageGen\|TestRegistryIncludesImage' -count=1`, `go run ./cmd/progress validate`
- Done signal: Hermetic tests prove image_generate can use a fake managed-gateway image provider and still emits the standard redacted artifact envelope and degraded evidence without live credentials.
- Acceptance: A fake managed gateway advertising an image-generation tool can be wrapped as an ImageGenProvider and selected by configured provider name or injected registry path., image_generate forwards prompt, model, aspect ratio, and output request data to the gateway, receives fixture image bytes/base64 or URL payload, and writes the standard redacted artifact envelope under a temp output directory., Auth-required, gateway-unavailable, schema-rejected, and tool-call-failed gateway outcomes return stable image-generation degraded evidence without registering half-discovered tools or leaking bearer tokens/prompts., Existing direct/fake image-generation provider behavior and managed gateway bridge tests continue to pass unchanged.
- Source refs: hermes-agent/tools/image_generation_tool.py, hermes-agent/tools/managed_tool_gateway.py, webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md: Image generation tool; MCP managed gateway, internal/tools/managed_tool_gateway.go, internal/tools/imagegen/generation.go, internal/tools/imagegen/provider.go, internal/tools/image_generation_provider.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
