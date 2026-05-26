import benchmarks from './benchmarks.json';
import release from './release.json';
import { absoluteDocsUrl, site } from './site.js';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const testCount = benchmarks?.code?.test_count || '';
const releaseVersion = release?.version || '0.1.01';
const releaseTag = release?.tag || `v${releaseVersion}`;
const releaseDateAlias = release?.date_alias || '';
const releaseLabel = releaseDateAlias
  ? `Current release: ${releaseTag} (${releaseDateAlias})`
  : `Current release: ${releaseTag}`;
const formattedTests = testCount ? testCount.toLocaleString() : '6,000+';

export const page = {
  siteUrl: site.url,
  title: 'Gormes — Go-Native AI Agent Runtime Without Python or Docker',
  description:
    'Gormes runs AI agents from one static Go binary with offline diagnostics, SQLite memory, provider chat, skills, dashboard, Telegram/Discord/Slack gateways, and an experimental Navivox app channel.',
  nav: [
    { label: 'Install', href: '/install' },
    { label: 'Docs', href: '/docs' },
    { label: 'Roadmap', href: '/roadmap' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  heroKicker: 'SINGLE BINARY AI AGENT RUNTIME',
  heroHeadline: 'Go-native AI agent runtime without Python or Docker.',
  heroLines: [
    'Run local and server-side AI agents from one static binary, with Hermes-style skills, offline diagnostics, SQLite memory, provider chat, dashboard, Telegram/Discord/Slack gateways, and an experimental Navivox app channel for trusted phone access.',
  ],
  primaryCta: { label: 'Install Gormes', href: '#install' },
  secondaryCta: {
    label: 'View GitHub',
    href: site.githubUrl,
  },
  proofStrip: [
    { label: 'Static Go binary', kind: 'pop' },
    { label: 'No venv drift', kind: 'pop' },
    { label: 'Offline doctor' },
    { label: 'Termux-ready release' },
  ],
  whyLabel: 'WHY GORMES',
  whyPainHeadline: 'Python agents break for boring reasons.',
  whyPainIntro:
    'Venvs drift, installs fail, streams drop, tools miswire, and servers rot. Gormes moves the runtime surface into one Go binary.',
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
      title: 'One gateway process',
      body: 'Telegram, Discord, Slack, and the experimental Navivox app channel use the same Go runtime and kernel path as local chat.',
    },
    {
      title: 'Static Go binary',
      body: 'Install or copy one binary across Linux, macOS, Windows, Termux, and locked-down servers.',
    },
    {
      title: 'Comparison matrix',
      body: 'Gormes is local software, not hosted zero-infrastructure SaaS; compare it with Hermes, OpenClaw, and hosted services on the operator axes buyers use.',
      linkLabel: 'Compare Gormes with Hermes, OpenClaw, and hosted services',
      href: absoluteDocsUrl('/why-gormes/#public-comparison-matrix'),
    },
  ],
  worksHeadline: 'What works today',
  worksIntro:
    'The current release is useful for local and server-side operator workflows without hiding the roadmap.',
  worksItems: [
    'CLI and offline TUI',
    'Provider-backed chat',
    'SQLite memory and sessions',
    'Local dashboard',
    'Telegram, Discord, Slack, and experimental Navivox channel',
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
    'A few public signals are enough for a landing page. The full roadmap and test evidence live in docs.',
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
    'More Hermes compatibility',
    'Voice/TTS',
    'MCP/plugin support',
    'Package-manager installs',
    'More gateways',
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
