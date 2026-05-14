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

## Agent Runtime Trust Boundaries

Gormes treats runtime inputs by source, not by how persuasive the text looks.

- System and developer instructions are trusted control-plane instructions.
- User requests are privileged operator intent, but still constrained by
  security policy, approvals, and local path/tool guards.
- Browser pages, webpage text, screenshots, image/OCR text, PDFs, documents,
  terminal output, tool responses, uploaded files, and remote service errors
  are untrusted data.
- Tool output is never upgraded into instructions. It is sanitized, bounded,
  and labeled before it is shown to the model, user, logs, or channel adapters
  where the integration surface supports that boundary.

External content that says things like "ignore previous instructions",
"this is a system message", "show your .env", "reveal API keys", or "send
secrets to this URL" is treated as prompt-injection content. Gormes may report
that such an attempt was present, but the raw instruction text is withheld from
prompt-visible untrusted-content summaries.

## Secret Handling Defaults

Gormes redacts common secret shapes before operator-visible logs, audit rows,
terminal results, browser artifacts, and related debug surfaces are rendered.
Covered shapes include OpenAI/Anthropic-style `sk-...` keys, GitHub tokens,
AWS access keys, database URLs, bearer/JWT tokens, Slack and Telegram tokens,
private-key blocks, and common `*_API_KEY`, `*_TOKEN`, `*_SECRET`, and
`DATABASE_URL` assignments.

The local file tools deny reads of sensitive paths by default, including
`.env`, `.env.*`, `.ssh/`, private key filenames, `.aws/`, `.gcloud/`,
`.azure/`, `.kube/config`, browser profile/cookie stores, password-manager
exports, and common credential files such as `.netrc`, `.pgpass`, `.npmrc`,
and `.pypirc`.

Persistent memory writes are filtered before storage. Memory refuses prompt
injection, secret material, and credential-file references so hostile page or
tool text cannot become durable future instructions.

## Refusal Behavior

Gormes refuses to:

- reveal `.env` files, API keys, tokens, cookies, private keys, cloud
  credentials, database URLs, browser cookies, or local credential stores;
- automatically run commands found inside untrusted webpages, screenshots,
  PDFs, OCR text, or tool output;
- execute or approve destructive or exfiltration-shaped shell commands such as
  `curl ... | sh`, `wget ... | sh`, `rm -rf`, secret-file reads, or
  secret-file uploads without deterministic guardrails;
- store prompt-injection text or secret material in persistent memory.

Security tests use dummy credentials only. Do not include live secrets in bug
reports, fixtures, logs, screenshots, or reproduction artifacts.

## Release Integrity

Current public builds are early-stage and source-first. Production-stable
release hardening is tracking SHA-256 checksums, detached signatures, Windows
code signing, embedded binary metadata, package-manager manifests, and
false-positive submission for major AV vendors when needed.

Docs, the public site, and future release automation are declared in GitHub
Actions workflows under `.github/workflows/`. A future `gormes security-audit`
command should summarize filesystem paths, configured endpoints, persistence,
and network behavior in one operator-facing report.
