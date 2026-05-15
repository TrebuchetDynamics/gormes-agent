import benchmarks from './benchmarks.json';
import release from './release.json';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const binaryMeasuredAt = benchmarks?.binary?.last_measured || '';
const runtimePeakRSSMB = benchmarks?.runtime?.offline_doctor?.peak_rss_mb || '';
const runtimeMeasuredAt = benchmarks?.runtime?.offline_doctor?.last_measured || '';
const goFiles = benchmarks?.code?.go_files || '';
const goLines = benchmarks?.code?.go_lines || '';
const testCount = benchmarks?.code?.test_count || '';
const platformCount = benchmarks?.properties?.platforms?.length || 0;
const releaseVersion = release?.version || '0.1.01';
const releaseTag = release?.tag || `v${releaseVersion}`;
const releaseDateAlias = release?.date_alias || '';
const releaseLabel = releaseDateAlias
  ? `Current scout release: ${releaseTag} (${releaseDateAlias})`
  : `Current scout release: ${releaseTag}`;
const binaryMeasureLabel = binarySizeMB
  ? `Current measured Linux build: ~${binarySizeMB} MB${binaryMeasuredAt ? ` (${binaryMeasuredAt})` : ''}`
  : 'Current Linux build measured during release prep';
const runtimeRSSLabel = runtimePeakRSSMB
  ? `Offline doctor peak RSS: ~${runtimePeakRSSMB} MB${runtimeMeasuredAt ? ` (${runtimeMeasuredAt})` : ''}`
  : 'Offline doctor peak RSS measured during release prep';
const codeBaseLabel = goFiles && goLines && testCount
  ? `${goFiles} Go files · ${Math.round(goLines / 1000)}k lines · ${testCount} tests`
  : '';

export const page = {
  title: 'Gormes — Run AI agents from a single binary',
  description:
    `One Go binary runs 30 Hermes skills on Termux, Windows, and locked-down Linux. No Python, no Docker, no dependency drift. Local SQLite memory, Telegram/Discord/Slack gateways, and an offline TUI — current Linux build ~${binarySizeMB || '40'} MB.`,
  nav: [
    { label: 'Docs', href: 'https://docs.gormes.ai/' },
    { label: 'Install', href: '#install' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  heroKicker: 'GO-NATIVE · OFFLINE-FIRST · MIT LICENSE',
  heroHeadline: 'Run AI agents from a single binary.',
  heroLines: [
    'One static binary. No Python runtime. No Docker daemon. No dependency drift.',
  ],
  heroFilterStamp: 'Scout release.',
  heroFilterLine:
    'Offline TUI, onboarding, provider turns, local SQLite memory, dashboard, and Telegram/Discord/Slack gateway paths are available now. Release signing, voice/TTS, and full Hermes parity are still hardening.',
  primaryCta: { label: 'Install', href: '#install' },
  secondaryCta: {
    label: 'See features',
    href: '#built-for',
  },
  proofStrip: [
    { label: '30 Hermes skills', kind: 'pop' },
    { label: '1 Go binary', kind: 'pop' },
    { label: `${testCount.toLocaleString()} tests`, kind: 'pop' },
    { label: 'MIT License' },
  ],
  methodologyLabel: 'HOW IT\'S BUILT',
  methodologyHeadline: 'Systematic porting with full test coverage.',
  methodologyIntro:
    "Every generated change passes tests, parity checks, and repo validation before landing. Hermes is the parity oracle; engineering rigor is the differentiator.",
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
    {
      label: 'Runtime RSS',
      value: runtimePeakRSSMB ? `~${runtimePeakRSSMB} MB` : 'measured at release',
      detail: runtimeRSSLabel,
    },
  ],
  methodologyPillars: [
    {
      title: 'Validation-gated commits',
      body: 'Every commit must pass go test, progress validate, and git diff --check before landing. No silent failures, no skipped hooks.',
    },
    {
      title: 'Hermes is the parity oracle',
      body: 'Upstream Hermes is the Python reference. The loop sweeps for gaps and turns them into test-driven slices.',
    },
  ],
  methodologyLink: {
    label: 'Read how the loop works ->',
    href: 'https://docs.gormes.ai/building-gormes/architecture_plan/',
  },
  installHeadline: 'Install first. Build from source when needed.',
  installIntro:
    'Use install.sh for the release-first managed install. Build from source when you need local inspection, custom flags, or unsupported platform fallback. Both paths keep the first proof offline.',
  installSteps: [
    {
      label: 'METHOD 1 · INSTALL.SH',
      command:
        'curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh\ngormes --version\ngormes doctor --offline',
    },
    {
      label: 'METHOD 2 · BUILD FROM SOURCE',
      command:
        'git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmkdir -p bin\nCGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes\n./bin/gormes doctor --offline\n./bin/gormes --offline',
    },
  ],
  installFootnote:
    'Use install.sh for the published gormes command on PATH, or ./bin/gormes from a source checkout when you are developing Gormes itself.',
  installFootnoteLink: {
    label: 'Read the install docs ->',
    href: 'https://docs.gormes.ai/using-gormes/install/',
  },
  afterProofHeadline: 'After setup',
  afterProofItems: [
    {
      label: 'Add a provider',
      command: 'gormes setup provider',
      body: 'Configure endpoint credentials only after the local doctor and offline TUI prove the machine.',
    },
    {
      label: 'Start chat',
      command: 'gormes chat',
      body: 'Open a provider-backed terminal chat before starting longer local sessions or gateways.',
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
      body: 'Developers and operators who need reliable, local agent infrastructure that survives restarts, bad networks, and dependency drift.',
    },
    {
      label: 'Not for yet',
      body: 'Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.',
    },
  ],
  trustHeadline: 'Trust posture',
  trustItems: [
    'Offline doctor runs before any token spend.',
    'Secrets stay local under ~/.gormes.',
    'Release-first install.sh you can inspect before running.',
    'Every commit passes go test, progress validate, and git diff --check.',
    'Tagged releases with SHA-256 checksums.',
  ],
  builtForHeadline: 'What works today',
  builtForGroups: [
    {
      title: 'Runtime',
      items: [
        'Offline TUI with zero dependencies',
        'One-shot provider turns',
        'Built-in environment doctor',
      ],
    },
    {
      title: 'Memory & State',
      items: [
        'Local SQLite sessions ("Goncho")',
        'Durable context across restarts',
        'Session browser and debug tools',
      ],
    },
    {
      title: 'Gateways',
      items: [
        'Telegram bot integration',
        'Discord bot integration',
        'Slack app integration',
      ],
    },
    {
      title: 'Operations',
      items: [
        'Local dashboard at 127.0.0.1:43827',
        'Security and secrets audit',
        'Gateway logs and status',
      ],
    },
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
        'Release-first install.sh and setup handoff',
        'Onboard/setup flows',
        'Provider-backed chat',
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
    { label: 'Built with', href: '/built-with' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  finalCtaHeadline: 'Prove the runtime locally before you ever spend a token.',
  finalCtaBody:
    'Run the release-first installer or build from source, run the offline doctor, then add credentials only after the machine has proven itself.',
  finalPrimaryCta: { label: 'Install now', href: '#install' },
  finalSecondaryCta: {
    label: 'Star on GitHub',
    href: 'https://github.com/TrebuchetDynamics/gormes-agent',
  },
  footerNav: [
    { label: 'Docs', href: 'https://docs.gormes.ai/' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
    { label: 'Company', href: 'https://trebuchetdynamics.com/' },
  ],
  footerRelease: releaseLabel,
};
