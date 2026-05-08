import benchmarks from './benchmarks.json';
import release from './release.json';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const binaryMeasuredAt = benchmarks?.binary?.last_measured || '';
const goFiles = benchmarks?.code?.go_files || '';
const goLines = benchmarks?.code?.go_lines || '';
const testCount = benchmarks?.code?.test_count || '';
const releaseVersion = release?.version || '0.1.01';
const releaseLabel = `Current scout release: v${releaseVersion}`;
const binaryMeasureLabel = binarySizeMB
  ? `Current measured Linux build: ~${binarySizeMB} MB${binaryMeasuredAt ? ` (${binaryMeasuredAt})` : ''}`
  : 'Current Linux build measured during release prep';
const codeBaseLabel = goFiles && goLines && testCount
  ? `${goFiles} Go files · ${Math.round(goLines / 1000)}k lines · ${testCount} tests`
  : '';

export const page = {
  title: 'Gormes — Hermes-Compatible AI Agent Runtime in One Go Binary',
  description:
    'Run Hermes-style AI agent skills, providers, memory, and gateways from one Go binary. No Python runtime. Built by an autonomous engineering loop that ports upstream Hermes daily under TDD discipline.',
  nav: [
    { label: 'How it is built', href: '#methodology' },
    { label: 'Install', href: '#install' },
    { label: 'Trust', href: '#trust' },
    { label: 'Roadmap', href: '#roadmap' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  heroKicker: 'OPEN SOURCE · MIT LICENSE · AUTONOMOUSLY PORTED FROM HERMES',
  heroHeadline: 'AI agent runtime in one Go binary.',
  heroLines: [
    'Drop-in compatible with the Hermes agent ecosystem — skills, providers, memory, and gateways — without a Python runtime, virtualenv, or backend service.',
    "Built by TrebuchetDynamics' autonomous engineering loop, which ports upstream Hermes daily under TDD discipline.",
    'Choose source build or install.sh, prove the machine offline, then add provider and gateway credentials.',
  ],
  heroFilterStamp: 'Scout release.',
  heroFilterLine:
    'Offline TUI, onboarding, provider turns, local SQLite memory, dashboard, and Telegram/Discord/Slack gateway paths are available now. Release signing, voice/TTS, and full Hermes parity are still hardening.',
  primaryCta: { label: 'Choose an install path', href: '#install' },
  secondaryCta: {
    label: 'See how it is built',
    href: '#methodology',
  },
  proofStrip: [
    'Source build recommended',
    'install.sh available',
    releaseLabel,
    'Static Go binary',
    'MIT License',
    'Offline doctor before credentials',
    'Autonomous porting loop ships daily',
  ],
  methodologyLabel: 'HOW IT IS BUILT',
  methodologyHeadline: 'An autonomous engineering loop ports Hermes to Go, every day.',
  methodologyIntro:
    "Gormes is the artifact TrebuchetDynamics' agentic engineering system produces. A planner → builder → TDD-slice loop runs against upstream Hermes around the clock, lands one bounded vertical slice at a time, and only commits when go test, progress validate, and git diff --check are all green. The methodology is the differentiator; Hermes-parity is the receipt that proves it works.",
  methodologyMetricLabel: 'Loop output, measured today',
  methodologyMetrics: [
    {
      label: 'Validated rows shipped',
      value: '770+',
      detail: 'Each carries a contract, fixtures, and acceptance evidence.',
    },
    {
      label: 'Code base',
      value: codeBaseLabel || `${goFiles} Go files`,
      detail: 'Static binary, zero CGO, no dynamic library deps.',
    },
    {
      label: 'Binary',
      value: binarySizeMB ? `~${binarySizeMB} MB` : 'measured at release',
      detail: 'Linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 today.',
    },
  ],
  methodologyPillars: [
    {
      title: 'Validation-gated commits',
      body: 'Every loop commit must pass go test ./..., go run ./cmd/progress validate, and git diff --check before landing on development. No silent failures, no skipped hooks.',
    },
    {
      title: 'progress.json as system of record',
      body: 'A schema-validated, contract-typed plan tracks every bounded slice. The loop selects the next builder-ready row; nothing else. No side queues, no private TODOs.',
    },
    {
      title: 'Hermes is the parity oracle',
      body: 'Upstream Hermes is the Python reference. The loop sweeps each release for behavior gaps, classifies them, and turns them into TDD slices the builder can finish.',
    },
    {
      title: 'Reusable porting toolkit',
      body: 'The skill set behind the loop (planner, builder, tdd-slice, parity-auditor, references, skill-manager) is generic Python-to-Go porting infrastructure. Open-source extraction is on the Q2 roadmap.',
    },
  ],
  methodologyLink: {
    label: 'Read how the loop works ->',
    href: 'https://docs.gormes.ai/building-gormes/architecture_plan/',
  },
  installHeadline: 'Three install paths. One gormes command.',
  installIntro:
    'Build from source when you want maximum inspection. Use install.sh on Linux/macOS/WSL2 or install.ps1 on native Windows when you want a source-backed managed install that publishes the stable gormes command. All paths keep the first proof offline. No runtime Node or npm is needed to open the native UI.',
  installSteps: [
    {
      label: 'METHOD 1 · BUILD FROM SOURCE',
      command:
        'git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build\nexport PATH="$PWD/bin:$PATH"\ngormes doctor --offline\ngormes --offline',
    },
    {
      label: 'METHOD 2 · INSTALL.SH (LINUX, MACOS, WSL2)',
      command:
        'curl -fsSLO https://gormes.ai/install.sh\nless install.sh\nsh install.sh\ngormes doctor --offline',
    },
    {
      label: 'METHOD 3 · INSTALL.PS1 (NATIVE WINDOWS)',
      command:
        'irm https://gormes.ai/install.ps1 -OutFile install.ps1\nGet-Content .\\install.ps1\npowershell -ExecutionPolicy Bypass -File .\\install.ps1\ngormes doctor --offline',
    },
  ],
  installFootnote:
    'All paths end at the same gormes command. install.sh and install.ps1 also run gormes setup when a terminal is available.',
  installFootnoteLink: {
    label: 'Read the install docs ->',
    href: 'https://docs.gormes.ai/using-gormes/install/',
  },
  afterProofHeadline: 'After offline proof',
  afterProofItems: [
    {
      label: 'Add a provider',
      command: 'gormes setup provider',
      body: 'Configure endpoint credentials only after the local doctor and offline TUI prove the machine.',
    },
    {
      label: 'Smoke-test a turn',
      command: 'gormes --oneshot "hello"',
      body: 'Run a single provider turn before starting longer local sessions or gateways.',
    },
    {
      label: 'Check gateway state',
      command: 'gormes gateway status',
      body: 'Promote Telegram, Discord, or Slack only after the configured channel reports clean status.',
    },
  ],
  fitHeadline: 'Who this is for',
  fitCards: [
    {
      label: 'For',
      body: 'Developers and operators who want local, inspectable agent infrastructure that survives restarts, bad networks, and dependency drift — and engineering teams curious how an autonomous loop ships parity work daily.',
    },
    {
      label: 'Not for yet',
      body: 'Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.',
    },
  ],
  trustHeadline: 'Trust posture',
  trustItems: [
    'Source build, inspectable install.sh (Linux/macOS/WSL2), and install.ps1 (native Windows) are the three promoted scout-release paths.',
    'Offline doctor runs before provider credentials or token spend.',
    'Secrets stay local under the Gormes home, not in the landing workflow.',
    'install.sh and install.ps1 clone or update a managed source checkout, build gormes, verify the command, and can hand off to setup.',
    'Tagged artifacts carry checksums; release signing and package-manager hardening are still in progress.',
    'Every autonomous-loop commit passes a validation gate (go test, progress validate, git diff --check) before landing.',
    binaryMeasureLabel,
    'Progress and benchmark data sync from repo sources during every landing build.',
  ],
  builtForHeadline: 'What works today',
  builtForItems: [
    'Run a local agent UI with zero runtime dependencies on the offline path',
    'Send one-shot prompts to a provider-compatible endpoint',
    'Validate your environment before spending tokens',
    'Run onboard/setup flows that surface config, providers, skills, agents, and channel bindings',
    'Operate Telegram, Discord, and Slack paths from one binary when configured',
    'Inspect and debug local SQLite memory ("Goncho")',
    'Browse sessions, config, skills, logs, and audits from local operator surfaces',
  ],
  supportHeadline: 'Gateway support status',
  supportRows: [
    {
      status: 'Runtime-ready',
      body: 'Telegram, Discord, and Slack.',
    },
    {
      status: 'In roadmap validation',
      body: 'WhatsApp, WeChat, Signal, Matrix, and Mattermost.',
    },
  ],
  whyLabel: 'WHY GORMES',
  whyPainHeadline: 'Python-stack agents fail for boring reasons.',
  whyPainIntro: 'The model is not usually the fragile part. Operations are:',
  whyPainBullets: [
    'dev, staging, and prod stop matching',
    'virtualenvs and package wheels drift across hosts',
    'long turns die on dropped streams',
    'tool wiring fails after tokens are already burning',
  ],
  whyFixSubhead: 'Gormes cuts out that failure class',
  featureCards: [
    {
      title: 'Single Binary Runtime',
      body: 'Static Go builds keep the runtime surface in one binary with no Python runtime dependency.',
    },
    {
      title: 'Offline Proof',
      body: 'gormes --offline starts the native TUI without credentials, network calls, Python, Node, Docker, or Hermes.',
    },
    {
      title: 'Built-In Doctor',
      body: 'gormes doctor --offline checks local readiness before provider calls or token spend.',
    },
    {
      title: 'Provider Turns',
      body: 'One-shots and the TUI use configured provider-compatible endpoints from the same binary.',
    },
    {
      title: 'Local SQLite Memory',
      body: 'Sessions, durable context, diagnostics, and queue state stay in local SQLite.',
    },
    {
      title: 'Visible Limits',
      body: 'Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening remain in progress.',
    },
  ],
  roadmapLabel: 'CURRENT STATE',
  roadmapHeadline: 'Core runtime shipped. Production hardening and broader parity are in progress.',
  roadmapBuckets: [
    {
      title: 'Shipped in scout',
      items: [
        'Offline TUI and doctor',
        'Source-backed install.sh and setup handoff',
        'Onboard/setup flows',
        'Provider one-shots',
        'Local SQLite memory and sessions',
        'Dashboard inspection',
        'Logs, security audit, and secrets audit',
        'Telegram, Discord, and Slack configured paths',
      ],
    },
    {
      title: 'Hardening now',
      items: [
        'Provider routing and auth edges',
        'Learning loop and operator feedback paths',
        'Tool safety and sandboxing',
        'Browser/web tools',
        'Release checksums, signing, and package-manager lanes',
      ],
    },
    {
      title: 'Later',
      items: [
        'Voice/TTS parity',
        'Broad channel parity',
        'Plugin/MCP parity',
        'Enterprise release polish',
      ],
    },
  ],
  roadmapNextMilestone:
    'Production-stable Go-native runtime with signed releases and broader Hermes parity',
  roadmapLinkLabel: 'View full roadmap ->',
  progressTrackerUrl: 'https://docs.gormes.ai/building-gormes/architecture_plan/',
  exploreHeadline: 'Explore',
  exploreLinks: [
    { label: 'Quickstart', href: 'https://docs.gormes.ai/using-gormes/quickstart/' },
    { label: 'Install', href: 'https://docs.gormes.ai/using-gormes/install/' },
    { label: 'Configuration', href: 'https://docs.gormes.ai/using-gormes/configuration/' },
    { label: 'Architecture', href: 'https://docs.gormes.ai/building-gormes/architecture_plan/' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  finalCtaHeadline: 'Prove the runtime locally before you ever spend a token.',
  finalCtaBody:
    'Build from source or inspect install.sh, run the offline doctor, then add credentials only after the machine has proven itself.',
  finalPrimaryCta: { label: 'Pick an install path', href: '#install' },
  finalSecondaryCta: {
    label: 'Star on GitHub',
    href: 'https://github.com/TrebuchetDynamics/gormes-agent',
  },
  footerNav: [
    { label: 'Docs', href: 'https://docs.gormes.ai/' },
    { label: 'Company', href: 'https://trebuchetdynamics.com/' },
  ],
  footerRelease: releaseLabel,
};
