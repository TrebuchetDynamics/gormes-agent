import benchmarks from './benchmarks.json';
import release from './release.json';
import { absoluteDocsUrl, site } from './site.js';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const testCount = benchmarks?.code?.test_count || '';
const releaseVersion = release?.version || '0.1.01';
const releaseTag = release?.tag || `v${releaseVersion}`;
const releaseDateAlias = release?.date_alias || '';
const releaseUrl = release?.url || `${site.githubUrl}/releases/latest`;
const releaseLabel = releaseDateAlias
  ? `Current release: ${releaseTag} (${releaseDateAlias})`
  : `Current release: ${releaseTag}`;
const formattedTests = testCount ? testCount.toLocaleString() : '6,000+';

export const page = {
  siteUrl: site.url,
  title: 'Gormes — Go AI Agent Runtime in One Static Binary',
  description:
    'Gormes is a local-first Go AI agent runtime for running local or server-side agents from one static binary with offline diagnostics, SQLite memory, provider chat, trusted gateways, and optional Navivox mobile pairing.',
  nav: [
    { label: 'Install', href: '/install' },
    { label: 'Docs', href: '/docs' },
    { label: 'Roadmap', href: '/roadmap' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  heroKicker: 'SINGLE BINARY AI AGENT RUNTIME',
  heroHeadline: 'Go-native AI agent runtime without Python or Docker',
  heroLines: [
    'Run local or server-side AI agents from one static binary with offline diagnostics, SQLite memory, provider chat, trusted gateways, and optional Navivox mobile pairing.',
  ],
  heroStatus:
    'Early-stage, useful today for CLI/TUI, provider turns, local state, and trusted gateways. Roadmap work continues on voice/TTS, more gateways, MCP/plugins, and deeper Hermes compatibility.',
  heroPanelEyebrow: 'OPERATOR QUICK PATH',
  heroPanelTitle: 'Install, verify, configure, chat.',
  heroPanelSteps: [
    {
      command: 'curl -fsSL https://gormes.ai/install.sh | bash',
      note: 'Install the current release-first binary.',
    },
    {
      command: 'gormes doctor --offline',
      note: 'Verify the machine before adding credentials.',
    },
    {
      command: 'gormes setup',
      note: 'Configure provider and runtime.',
    },
    {
      command: 'gormes chat',
      note: 'Start provider-backed chat.',
    },
  ],
  heroMetrics: [
    { value: binarySizeMB ? `~${binarySizeMB} MB` : '~46 MB', label: 'Linux binary' },
    { value: formattedTests, label: 'tests' },
    { value: 'offline', label: 'doctor first' },
  ],
  primaryCta: { label: 'Install Gormes', href: '#install' },
  secondaryCta: {
    label: 'View GitHub',
    href: site.githubUrl,
  },
  tertiaryCta: {
    label: 'See install docs',
    href: '/install',
  },
  proofStrip: [
    { label: 'Static Go binary', kind: 'pop' },
    { label: 'No venv drift', kind: 'pop' },
    { label: 'Offline doctor' },
    { label: 'Android/Termux-ready release' },
  ],
  whyLabel: 'WHY GORMES',
  whyPainHeadline: 'Python agents break for boring reasons.',
  whyPainIntro:
    'Venvs drift, installs fail, streams drop, tools miswire, and servers rot. Gormes moves the runtime surface into one Go binary you can inspect and move.',
  whyCards: [
    {
      title: 'No venv drift',
      body: 'No pip, virtualenv, Python wheel, Node sidecar, or Docker daemon on the core runtime path.',
    },
    {
      title: 'Offline doctor first',
      body: '`gormes doctor --offline` checks local readiness before credentials or provider token spend.',
    },
    {
      title: 'Local SQLite memory',
      body: 'Sessions, durable context, diagnostics, and recall stay inspectable under the local Gormes home.',
    },
    {
      title: 'Trusted gateways',
      body: 'Telegram, Discord, Slack, and Navivox connect through the same Go runtime and local control plane.',
    },
    {
      title: 'Portable runtime',
      body: 'Install or copy one binary across Linux, macOS, Windows, WSL2, VPS hosts, home servers, and Android/Termux.',
    },
    {
      title: 'Hermes-style skills',
      body: 'The runtime keeps skills and tools in the Go process while the roadmap tracks remaining Hermes compatibility work.',
    },
  ],
  saasHeadline: 'Why not hosted SaaS?',
  saasIntro:
    'Gormes is for operators who want a local-first runtime they can inspect, move, and connect to their own providers.',
  saasItems: [
    'Local-first control plane',
    'Offline readiness checks',
    'Your own provider keys',
    'Portable one-binary deploys',
    'No hosted bot lock-in',
    'Inspectable state and logs',
  ],
  worksHeadline: 'What works today',
  worksIntro:
    'The current release is useful for local and server-side operator workflows without hiding the roadmap.',
  localRemoteCopy:
    'You can run Gormes locally on your machine or remotely on a VPS/home server and manage it through CLI, gateways, or Navivox.',
  navivoxCopy:
    'Navivox is an experimental mobile control app for pairing with local or remote Gormes runtimes.',
  worksItems: [
    'CLI and offline TUI',
    'Provider-backed chat',
    'SQLite memory and sessions',
    'Local dashboard',
    'Telegram, Discord, Slack, and experimental Navivox channel',
  ],
  runTargets: ['Linux', 'macOS', 'Windows', 'WSL2', 'VPS', 'Home servers', 'Android/Termux'],
  workflowExamples: [
    'Telegram assistant',
    'Local coding agent',
    'Persistent memory assistant',
    'VPS runtime managed from phone',
  ],
  runtimeVisualHeadline: 'Runtime view',
  runtimeVisualIntro:
    'The screenshot below is generated from a real `gormes --offline` TUI run captured in tmux, not a mockup.',
  runtimeVisuals: [
    {
      title: 'CLI/TUI operator loop',
      body: 'Captured from the committed Gormes runtime in offline smoke-test mode; the TUI runs locally from the same binary used by chat, doctor, setup, and gateway commands.',
      image: '/static/gormes-tui-offline-capture.png',
      alt: 'Real Gormes offline TUI capture showing the local operator interface',
      width: 1280,
      height: 720,
    },
  ],
  installHeadline: 'Install Gormes',
  installIntro:
    'Use the release installer, verify the binary, prove the machine offline before credentials, then add a provider and start chat. No pip, no venv, no Docker daemon.',
  installCommand:
    'curl -fsSL https://gormes.ai/install.sh | bash\ngormes version\ngormes doctor --offline\ngormes setup\ngormes chat',
  installFootnote:
    'Termux/Android status: v0.2.23 carries forward the installer recovery for the v0.2.20 executable-argument bug; affected users should reinstall the latest release, then verify with gormes version and gormes doctor --offline. Windows, source builds, and advanced installer flags are covered in the install docs.',
  installFootnoteLink: {
    label: 'Read install docs',
    href: '/install',
  },
  proofHeadline: 'Evidence, not a sidecar stack',
  proofIntro:
    'A few public signals are enough for a landing page. The full roadmap, release assets, and benchmark evidence live in docs and GitHub.',
  proofItems: [
    {
      value: formattedTests,
      label: 'tests',
      detail: 'CI also gates progress validation and whitespace checks.',
    },
    {
      value: binarySizeMB ? `~${binarySizeMB} MB` : '~46 MB',
      label: 'Linux binary',
      detail: 'Measured from the release benchmark mirror.',
    },
    {
      value: 'offline',
      label: 'doctor before credentials',
      detail: 'Local readiness can be checked without credentials, network calls, Python, Node, npm, or Docker.',
    },
    {
      value: 'SHA-256 + SBOM',
      label: 'release assets',
      detail: 'Tagged releases publish checksums and SBOMs.',
    },
  ],
  proofLinks: [
    {
      label: 'pkg.go.dev',
      href: 'https://pkg.go.dev/github.com/TrebuchetDynamics/gormes-agent/pkg/gormes',
    },
    {
      label: 'Latest GitHub release',
      href: releaseUrl,
    },
    {
      label: 'Install and release docs',
      href: absoluteDocsUrl('/install/'),
    },
    {
      label: 'Benchmark data',
      href: `${site.githubUrl}/blob/development/benchmarks.json`,
    },
    {
      label: 'Memory and retrieval evidence',
      href: absoluteDocsUrl('/architecture/memory-and-sessions/'),
    },
    {
      label: 'Goncho retrieval benchmark',
      href: absoluteDocsUrl('/building-gormes/architecture_plan/#goncho-retrieval-benchmark-corpus'),
    },
    {
      label: 'Comparison matrix',
      href: absoluteDocsUrl('/why-gormes/#public-comparison-matrix'),
    },
  ],
  roadmapLabel: 'ROADMAP',
  roadmapHeadline: 'Available now. Expanding next.',
  roadmapNow: [
    'CLI/TUI',
    'Provider chat',
    'SQLite memory',
    'Dashboard',
    'Telegram/Discord/Slack',
    'Navivox pairing',
  ],
  roadmapNext: [
    'Voice/TTS',
    'More gateways',
    'MCP/plugin support',
    'More Hermes compatibility',
  ],
  roadmapLinkLabel: 'View the roadmap',
  progressTrackerUrl: '/roadmap',
  finalCtaHeadline: 'Run the offline doctor before you spend a token.',
  finalCtaBody:
    'Install Gormes, prove the runtime locally, then configure a provider and gateway when the machine is ready.',
  finalPrimaryCta: { label: 'Run offline doctor', href: '#install' },
  finalSecondaryCta: {
    label: 'Star on GitHub',
    href: site.githubUrl,
  },
  footerNav: [
    { label: 'Docs', href: '/docs' },
    { label: 'Install', href: '/install' },
    { label: 'Roadmap', href: '/roadmap' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  releaseTag,
  footerRelease: releaseLabel,
};
