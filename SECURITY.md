# Security Policy

Gormes is an AI-agent runtime that performs local file I/O, provider network
calls, process orchestration, and gateway work. Please report vulnerabilities
privately before opening public issues.

## Reporting

Preferred: use GitHub private vulnerability reporting for this repository when
available.

Fallback: email `security@trebuchetdynamics.com` with:

- affected version or commit;
- operating system and install method;
- reproduction steps;
- expected impact;
- any logs needed to verify the report.

Do not include live API keys, private prompts, or sensitive local databases in
the report.

## Scope

In scope:

- command execution or path traversal bugs;
- unsafe installer behavior;
- credential exposure;
- unauthorized network calls;
- Goncho memory database leakage or corruption;
- gateway authentication and authorization flaws.

Out of scope:

- reports against dependencies with no Gormes-specific exploit path;
- social engineering;
- denial-of-service reports that require excessive traffic against public
  infrastructure;
- scanner-only reports without a reproducible security impact.

## Release Integrity

Current public builds are early-stage and source-first. Production-stable
release hardening is tracking SHA-256 checksums, detached signatures, Windows
code signing, embedded binary metadata, package-manager manifests, and
false-positive submission for major AV vendors when needed.
