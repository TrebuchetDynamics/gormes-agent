# README Quality Rubric

Use this reference when a `gormes-readme` pass needs a 100/100 repository
audit, README score, or broad public-surface improvement plan.

## Litmus Test

A strong repository lets a stranger quickly understand it, run it without pain,
trust its claims, and see how to contribute.

## Score Areas

Rate each area as `clear`, `partial`, or `missing`. Recommend README changes
only when they can be backed by current repository evidence.

1. **Purpose** - The first screen explains what Gormes is, why it exists, who it
   is for, and why the reader should care.
2. **README Front Door** - The README has a copy-paste quick start, install
   instructions that match the repo, feature/use-case summary, deeper-doc links,
   and a short TLDR-style opening.
3. **Developer Experience** - Setup, build, run, test, and diagnostics paths are
   discoverable and use existing repo commands such as `make`, `gormes doctor`,
   `cmd/README.md`, and install scripts.
4. **Codebase Orientation** - The README or linked docs show the logical project
   shape without duplicating full architecture docs.
5. **Documentation Depth** - Links exist for API/CLI usage, architecture,
   examples, troubleshooting, and roadmap/progress.
6. **Tests And Reliability** - The README points to meaningful validation:
   tests, CI when present, reproducible builds, error handling, and current
   reliability limits.
7. **Versioning And Releases** - Tags, release notes, changelog, or explicit
   early-stage status make adoption risk clear.
8. **Maintenance Signal** - The public surface makes the project feel alive
   without promising unsupported response times or production readiness.
9. **Contribution Path** - Contribution docs, issue/PR expectations, coding
   standards, and good-first-issue surfaces are findable when present.
10. **License And Governance** - License, ownership, lineage, and decision
    authority are clear enough that users know the legal and project context.
11. **Performance And Practical Value** - Benchmarks, binary size, comparisons,
    or performance claims are included only when measured and current.
12. **Polish** - Badges, formatting, visuals, demos, naming, links, and command
    snippets are consistent and not stale.

## Prioritization

Fix in this order:

1. Wrong or unsupported claims.
2. Missing run/install path.
3. Missing current status and limitations.
4. Missing trust signals: tests, license, docs, progress, releases.
5. Polish: badges, visuals, concise wording, and link hygiene.

Do not turn the README into the whole docs site. Link out when detail would make
the README harder to scan.
