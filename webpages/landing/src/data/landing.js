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
const binaryLabel = binarySizeMB ? `~${binarySizeMB} MB` : '~46 MB';

export const page = {
  siteUrl: site.url,
  title: 'Gormes — Run AI Agents Anywhere from One Go Binary',
  description:
    'Gormes is a Go-native AI agent runtime for local or server-side agents with offline diagnostics, SQLite memory, provider chat, dashboards, trusted gateways, and experimental Navivox phone pairing.',
  nav: [
    { label: 'Install', href: '/install' },
    { label: 'Docs', href: '/docs' },
    { label: 'Roadmap', href: '/roadmap' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  heroKicker: 'ONE GO BINARY',
  heroHeadline: 'Run AI agents anywhere.',
  heroLines: [
    'Gormes is a Go-native runtime for local or server-side agents: provider chat, SQLite memory, dashboards, and trusted gateways without Python, Docker, or venv drift.',
  ],
  heroStatus:
    'Works today for CLI/TUI, provider chat, memory, dashboards, Telegram/Discord/Slack, and experimental Navivox phone pairing.',
  heroPanelEyebrow: 'FIRST RUN',
  heroPanelTitle: 'Install. Verify offline. Start chat.',
  heroPanelSteps: [
    {
      command: 'curl -fsSL https://gormes.ai/install.sh | bash',
      note: 'Install the current release-first binary.',
    },
    {
      command: 'gormes doctor --offline',
      note: 'Check the machine before adding credentials.',
    },
    {
      command: 'gormes setup',
      note: 'Configure providers, runtime, and gateways.',
    },
    {
      command: 'gormes chat',
      note: 'Run a provider-backed agent turn.',
    },
  ],
  heroMetrics: [
    { value: binaryLabel, label: 'binary' },
    { value: formattedTests, label: 'tests' },
    { value: 'SHA-256 + SBOM', label: 'release proof' },
  ],
  primaryCta: { label: 'Install Gormes', href: '#install' },
  secondaryCta: { label: 'Read docs', href: '/docs' },
  tertiaryCta: { label: 'View GitHub', href: site.githubUrl },
  proofStrip: [
    { label: `${binaryLabel} binary`, kind: 'pop' },
    { label: `${formattedTests} tests`, kind: 'pop' },
    { label: 'Linux · macOS · Windows · VPS · Android' },
    { label: 'MIT' },
  ],
  whyLabel: 'WHY GORMES',
  whyPainHeadline: 'One binary. Less drift.',
  whyPainIntro:
    'Install once, verify offline, then run chat, memory, dashboards, and gateways.',
  whyCards: [
    {
      title: 'No Python drift',
      body: 'No pip, virtualenv, Python wheel, Node sidecar, or Docker daemon in the core path.',
    },
    {
      title: 'Offline doctor',
      body: '`gormes doctor --offline` checks readiness before credentials, network calls, or token spend.',
    },
    {
      title: 'SQLite memory',
      body: 'Sessions, context, diagnostics, and recall stay inspectable under the local Gormes home.',
    },
    {
      title: 'Trusted gateways',
      body: 'Telegram, Discord, Slack, and experimental Navivox pairing share the same Go runtime.',
    },
  ],
  installHeadline: 'First run',
  installIntro:
    'Install with no pip, no venv, and no Docker daemon. Verify offline before credentials and without credentials, network calls, Python, Node, or token spend, then start chat.',
  installCommand:
    'curl -fsSL https://gormes.ai/install.sh | bash\ngormes version\ngormes doctor --offline\ngormes setup\ngormes chat',
  installFootnote:
    'Windows, source builds, Termux details, and advanced installer flags are covered in the install docs.',
  installFootnoteLink: { label: 'Read install docs', href: '/install' },
  runtimeVisualHeadline: 'Real runtime, not a mockup',
  runtimeVisualIntro:
    'A real `gormes --offline` TUI run captured in tmux. Same binary: chat, doctor, setup, dashboards, gateways.',
  runtimeVisuals: [
    {
      title: 'CLI/TUI operator loop',
      body: 'Captured from the committed Gormes runtime in offline smoke-test mode.',
      image: '/static/gormes-tui-offline-capture.png',
      alt: 'Real Gormes offline TUI capture showing the local operator interface',
      width: 1280,
      height: 720,
    },
  ],
  useCasesHeadline: 'Use cases',
  useCasesIntro: 'Three simple ways to use one runtime.',
  useCases: [
    {
      title: 'Personal agent',
      body: 'Run a local coding, research, or memory assistant.',
    },
    {
      title: 'VPS agent',
      body: 'Keep a server-side runtime online and reach it through dashboards or chat gateways.',
    },
    {
      title: 'Phone-controlled runtime',
      body: 'Experimental: pair a phone with a local or remote Gormes runtime using Navivox.',
      href: absoluteDocsUrl('/cli/channels/navivox/'),
      linkLabel: 'Navivox docs',
    },
  ],
  proofHeadline: 'Proof you can verify',
  proofIntro:
    'Short trust signals. Full release, benchmark, and roadmap evidence lives in docs and GitHub.',
  proofItems: [
    {
      value: formattedTests,
      label: 'tests',
      detail: 'CI also gates progress validation and whitespace.',
    },
    {
      value: binaryLabel,
      label: 'static binary',
      detail: 'Measured from release benchmark data.',
    },
    {
      value: 'SHA-256 + SBOM',
      label: 'release assets',
      detail: 'Tagged releases publish checksums and SBOMs.',
    },
    {
      value: 'pkg.go.dev',
      label: 'public Go package',
      detail: 'Public Go API surface.',
    },
  ],
  proofLinks: [
    {
      label: 'pkg.go.dev',
      href: 'https://pkg.go.dev/github.com/TrebuchetDynamics/gormes-agent/pkg/gormes',
    },
    { label: 'Latest GitHub release', href: releaseUrl },
    { label: 'Install docs', href: absoluteDocsUrl('/install/') },
    { label: 'Roadmap', href: '/roadmap' },
  ],
  finalCtaHeadline: 'Install once. Verify offline. Run your agent.',
  finalCtaBody:
    'Start with the offline doctor. Add providers, memory, gateways, or Navivox when ready.',
  finalPrimaryCta: { label: 'Install Gormes', href: '#install' },
  finalSecondaryCta: { label: 'Star on GitHub', href: site.githubUrl },
  footerNav: [
    { label: 'Docs', href: '/docs' },
    { label: 'Install', href: '/install' },
    { label: 'Roadmap', href: '/roadmap' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  releaseTag,
  footerRelease: releaseLabel,
};
