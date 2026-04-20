---
title: "Technology Radar"
weight: 100
---

## 9. Technology Radar — Package & Tool Research

Continuous research into the Go ecosystem for Gormes-relevant packages, techniques, and upstream developments.

### 9.1 Vector Embedding Libraries (Phase 3.D Research — 2026-04-20)

Evaluated pure-Go vector databases for semantic recall layer:

| Library | License | Storage | Index | Size Impact | Notes |
|---------|---------|---------|-------|-------------|-------|
| **[chromem-go](https://github.com/TIANLI0/chromem-go)** | Apache-2.0 | In-memory + optional persist | HNSW, IVF, PQ, BM25, hybrid | ~200KB | Zero third-party deps; SIMD on amd64; BM25 for lexical+semantic fusion |
| **[veclite](https://github.com/abdul-hamid-achik/veclite)** | MIT | Single `.veclite` file | HNSW + BM25 | ~150KB | Zero deps (stdlib only); auto-embedding with Ollama/OpenAI; single-file portability |
| **[vecgo](https://github.com/hupe1980/vecgo)** | Apache-2.0 | Commit-oriented durability | HNSW + DiskANN/Vamana | ~300KB | Production-focused; 16-way sharded HNSW; arena allocator; PQ/RaBitQ quantization |
| **[govector](https://github.com/DotNetAge/govector)** | MIT | bbolt + Protobuf | HNSW | ~250KB | "SQLite for Vectors"; Qdrant-compatible API; uses `github.com/coder/hnsw` |
| **[goformersearch](https://github.com/MichaelAyles/goformersearch)** | MIT | In-memory | Brute-force + HNSW | ~100KB | Minimal surface; designed for 10k-50k docs at 384d; single-core optimized |

**Recommendation**: **chromem-go** or **veclite** for Phase 3.D. Both offer:
- Pure Go (CGO-free, static binary compatible)
- HNSW for O(log n) ANN search
- BM25 for hybrid lexical+semantic search
- Zero additional dependencies
- MIT/Apache-2.0 licenses (compatible with Gormes)

**Ollama Integration**: Ollama supports OpenAI-compatible `/v1/embeddings` endpoint. Go client libraries: [`go-embeddings`](https://github.com/milosgajdos/go-embeddings) (multi-provider, includes Ollama), [`go-ollama`](https://github.com/eslider/go-ollama) (streaming support).

### 9.2 SQLite Driver Landscape

Current: `github.com/ncruces/go-sqlite3` (WASM-based, CGO-free)

Alternatives monitored:
- `modernc.org/sqlite` (C-to-Go transpiled, larger binary impact)
- `github.com/mattn/go-sqlite3` (CGO, not static-binary friendly)

**Status**: ncruces driver remains optimal for CGO-free static builds.

### 9.3 Upstream Hermes-Agent Tracking

**Repository**: https://github.com/NousResearch/hermes-agent
**License**: MIT (compatible)
**Porting Strategy**: Strangler Fig — Gormes phases gradually subsume Python subsystems (§1 Rosetta Stone)

**Recent upstream additions to monitor** (inventory from parallel codebase audit):
- Gateway platforms: 24 adapters including Telegram, Discord, Slack, WhatsApp, Signal, Email, SMS, Feishu, Matrix, Weixin, BlueBubbles, QQ
- RL training environments (`environments/`)
- ACP adapter for IDE integration (`acp_adapter/`)
- Honcho dialectic user modeling integration
- Skills Hub registries (skills.sh, clawhub, lobehub, hermes-index)

**Cadence**: Re-run upstream survey on major Hermes releases or when new platform connectors land. See Subsystem Inventory for complete upstream file mapping.

---

*Technology Radar v1.0 — Research synthesized from web searches and parallel codebase audit.*
