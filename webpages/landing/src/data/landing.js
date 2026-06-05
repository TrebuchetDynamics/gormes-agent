import benchmarks from './benchmarks.json';
import release from './release.json';
import { site } from './site.js';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const testCount = benchmarks?.code?.test_count || '';
const releaseVersion = release?.version || '0.1.01';
const releaseTag = release?.tag || `v${releaseVersion}`;
const releaseDateAlias = release?.date_alias || '';
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
    { label: 'Install', href: '#install' },
    { label: 'Docs', href: '/docs' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  heroKicker: 'ONE GO BINARY',
  heroHeadline: 'Run AI agents anywhere.',
  heroLines: [
    'Gormes runs local or server-side agents with chat, memory, dashboards, and gateways — no Python, Docker, or venv drift.',
  ],
  primaryCta: { label: 'Install Gormes', href: '#install' },
  secondaryCta: { label: 'Read docs', href: '/docs' },
  tertiaryCta: { label: 'View GitHub', href: site.githubUrl },
  proofStrip: [
    { label: `${binaryLabel} static binary`, kind: 'pop' },
    { label: 'Linux · macOS · Windows · Android' },
    { label: 'MIT' },
  ],
  whyLabel: 'WHY GORMES',
  whyPainHeadline: 'Less setup. More runtime.',
  whyPainIntro: 'A small runtime you can inspect, move, and run without a language toolchain.',
  whyCards: [
    {
      title: 'No Python drift',
      body: 'No pip, virtualenv, Python wheel, Node sidecar, or Docker daemon.',
    },
    {
      title: 'Offline doctor',
      body: '`gormes doctor --offline` checks readiness before credentials or token spend.',
    },
    {
      title: 'SQLite memory',
      body: 'Sessions, context, diagnostics, and recall stay local and inspectable.',
    },
  ],
  installHeadline: 'Run it',
  installIntro:
    'No pip, no venv, no Docker daemon: run gormes version, then gormes doctor --offline before credentials and without credentials, network calls, Python, Node, or token spend.',
  installCommand:
    'curl -fsSL https://gormes.ai/install.sh | bash\ngormes version\ngormes doctor --offline\ngormes setup\ngormes chat',
  installFootnote:
    'Windows, source builds, Termux details, and advanced installer flags are covered in the install docs.',
  installFootnoteLink: { label: 'Read install docs', href: '/install' },
  footerNav: [
    { label: 'Docs', href: '/docs' },
    { label: 'Install', href: '/install' },
    { label: 'GitHub', href: site.githubUrl },
  ],
  releaseTag,
  footerRelease: releaseLabel,
};
