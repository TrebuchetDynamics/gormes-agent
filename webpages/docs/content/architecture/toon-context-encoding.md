---
title: "TOON context encoding"
description: "Optional compact serialization for model-visible prompt and context payloads."
weight: 46
---

# TOON context encoding

Gormes treats TOON as an optional serialization layer for JSON-shaped data that
is going into prompts or other model-visible context. It is not a replacement
for JSON in provider APIs, public APIs, config, or storage contracts.

The local implementation lives in `internal/toon`. Its public entry points are
JSON-first:

- `toon.EncodeJSON([]byte)` parses JSON while preserving object key order, then
  emits TOON.
- `toon.DecodeJSON([]byte)` parses TOON and returns compact JSON.
- `toon.Encode(any)` is a convenience for Go values that can already marshal to
  JSON.
- `toon.NewEncoder(io.Writer)` writes encoded TOON to a caller-owned writer.

`internal/hermes.EncodePromptContext` wires this in as an explicit
`PromptContextFormatTOON` option. The default prompt-context format remains
compact JSON.

## Good fits

Use TOON where the payload is JSON data and the consumer is an LLM prompt:

- prompt payloads assembled from structured context
- tool-result summaries before they are injected into context
- memory or session summaries
- large tabular context
- debuggable agent-state snapshots

TOON is most useful when repeated object arrays can be emitted as tabular rows,
or when JSON punctuation is dominating a model-visible payload.

## Keep JSON here

Do not use TOON for:

- provider HTTP request or response bodies
- public APIs that advertise JSON
- security-sensitive canonical signing, hashing, or replay comparison
- core persisted storage formats
- config files, unless the user explicitly chooses a `.toon` file format

Those surfaces need established JSON tooling, provider compatibility, and stable
canonical behavior. TOON should be a prompt/context encoding choice at the edge,
not the runtime's core data model.

## Conformance note

The `internal/toon` package preserves the JSON data model: object, array,
string, number, boolean, and null. It implements a conservative subset of the
TOON v3.1 working-draft grammar needed by Gormes: nested objects, counted
arrays, primitive inline arrays, primitive-object tabular arrays, list arrays,
quoted strings, strict tabular row counts, and JSON round trips.

If Gormes needs broader ecosystem interchange later, expand the package against
the upstream TOON spec and add fixtures before using new grammar.
