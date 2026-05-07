import benchmarks from './benchmarks.json';
import release from './release.json';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const binaryMeasuredAt = benchmarks?.binary?.last_measured || '';
const releaseVersion = release?.version || '0.1.01';
const releaseLabel = `Current scout release: v${releaseVersion}`;
const binaryMeasureLabel = binarySizeMB
  ? `Current measured Linux build: ~${binarySizeMB} MB${binaryMeasuredAt ? ` (${binaryMeasuredAt})` : ''}`
  : 'Current Linux build measured during release prep';

export const page = {
  title: 'Gormes — Run AI Agents From One Go Binary',
  description:
    'Gormes runs local agent sessions, provider turns, memory, dashboards, and chat gateways from one Go binary. Build from source, prove the machine offline, then add credentials.',
  nav: [
    { label: 'Docs', href: 'https://docs.gormes.ai/' },
    { label: 'Trust', href: '#trust' },
    { label: 'Roadmap', href: '#roadmap' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  heroKicker: 'OPEN SOURCE · MIT LICENSE · SCOUT RELEASE',
  heroHeadline: 'Run AI agents from one Go binary.',
  heroLines: [
    'Gormes runs local agent sessions, provider turns, memory, dashboards, and chat gateways from one Go binary.',
    'No Python runtime. No virtualenv repair. No backend service just to open the UI.',
    'Choose source build or install.sh, prove the machine offline, then add provider and gateway credentials.',
  ],
  heroFilterStamp: 'Scout release.',
  heroFilterLine:
    'Offline TUI, onboarding, provider turns, local SQLite memory, dashboard, and Telegram/Discord/Slack gateway paths are available now. Release signing, voice/TTS, and full Hermes parity are still hardening.',
  primaryCta: { label: 'Choose an install path', href: '#install' },
  secondaryCta: {
    label: 'View on GitHub',
    href: 'https://github.com/TrebuchetDynamics/gormes-agent',
  },
  proofStrip: [
    'Source build recommended',
    'install.sh available',
    releaseLabel,
    'Static Go binary',
    'MIT License',
    'Offline doctor before credentials',
  ],
  installHeadline: 'Two install paths. One gormes command.',
  installIntro:
    'Build from source when you want maximum inspection. Use install.sh when you want a source-backed managed install that publishes the stable gormes command. Both paths keep the first proof offline. No runtime Node or npm is needed to open the native UI.',
  installSteps: [
    {
      label: 'METHOD 1 · BUILD FROM SOURCE',
      command:
        'git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build\nexport PATH="$PWD/bin:$PATH"\ngormes doctor --offline\ngormes --offline',
    },
    {
      label: 'METHOD 2 · INSTALL.SH',
      command:
        'curl -fsSLO https://gormes.ai/install.sh\nless install.sh\nsh install.sh\ngormes doctor --offline',
    },
  ],
  installFootnote:
    'Both paths end at the same gormes command. install.sh also runs gormes setup when a terminal is available.',
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
      body: 'Developers and operators who want local, inspectable agent infrastructure that survives restarts, bad networks, and dependency drift.',
    },
    {
      label: 'Not for yet',
      body: 'Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.',
    },
  ],
  trustHeadline: 'Trust posture',
  trustItems: [
    'Source build and inspectable install.sh are the two promoted scout-release paths.',
    'Offline doctor runs before provider credentials or token spend.',
    'Secrets stay local under the Gormes home, not in the landing workflow.',
    'install.sh clones or updates a managed source checkout, builds gormes, verifies the command, and can hand off to setup.',
    'Tagged artifacts carry checksums; release signing and package-manager hardening are still in progress.',
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
