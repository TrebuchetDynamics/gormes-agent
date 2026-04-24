---
title: "Upstream GBrain Study"
weight: 350
---

# Upstream GBrain Study

This section studies upstream `gbrain` as an architecture donor for Gormes.
It is not a porting instruction to copy GBrain wholesale. The goal is to
extract the useful ideas, name the failure modes, and apply them to a better
Go-native `gormes-agent`.

## Study Snapshot

- Upstream studied: `/home/xel/git/sages-openclaw/workspace-mineru/gbrain`
- Upstream commit: `11abb24ddd2209f8622870c2e48dc9ef050ad749`
- Gormes repo studied: `/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent`
- Gormes commit: `8c173263eb7e13b7acd4c0e2145ede19e7a0a3f2`
- Date: 2026-04-24

## Documents

- [Architecture](./architecture/) maps the GBrain runtime, data model, search
  stack, skills layer, and Minions job system.
- [Good and Bad](./good-and-bad/) lists the design moves worth stealing and the
  traps Gormes should avoid.
- [Gormes Takeaways](./gormes-takeaways/) translates the study into concrete
  Gormes architecture decisions.

## One-Line Read

GBrain's best idea is not "Postgres brain." It is the combination of
contract-first operations, a brain-first agent loop, fat procedural skills, and
a durable job ledger for deterministic work. Gormes should keep the single Go
binary and typed tool contracts, then borrow those ideas in a smaller,
auditable, SQLite-first shape.

