import benchmarks from './benchmarks.json';
import progress from './progress.json';
import release from './release.json';

const STATUS_COMPLETE = 'complete';
const STATUS_IN_PROGRESS = 'in_progress';
const STATUS_PLANNED = 'planned';

const binarySizeMB = benchmarks?.binary?.size_mb || '';
const releaseVersion = release?.version || '0.1.01';
const releaseLabel = `Current scout release: v${releaseVersion}`;
const binaryMeasureLabel = binarySizeMB
  ? `Current measured Linux build: ~${binarySizeMB} MB`
  : 'Current Linux build measured during release prep';

function derivedSubphaseStatus(subphase) {
  const items = Array.isArray(subphase.items) ? subphase.items : [];
  if (items.length === 0) {
    return subphase.status || STATUS_PLANNED;
  }

  const allComplete = items.every((item) => item.status === STATUS_COMPLETE);
  const anyStarted = items.some(
    (item) => item.status === STATUS_COMPLETE || item.status === STATUS_IN_PROGRESS,
  );

  if (allComplete) {
    return STATUS_COMPLETE;
  }
  if (anyStarted) {
    return STATUS_IN_PROGRESS;
  }
  return STATUS_PLANNED;
}

export function progressTrackerLabel(source = progress) {
  const subphases = Object.values(source.phases || {}).flatMap((phase) =>
    Object.values(phase.subphases || {}),
  );
  const complete = subphases.filter(
    (subphase) => derivedSubphaseStatus(subphase) === STATUS_COMPLETE,
  ).length;
  return `${complete}/${subphases.length} shipped`;
}

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
    'Start offline, prove the machine works, then add provider and gateway credentials.',
  ],
  heroFilterStamp: 'Scout release: useful today, still early.',
  heroFilterLine:
    'Offline TUI, doctor diagnostics, provider one-shots, local SQLite memory, dashboard, and runtime-ready Telegram/Discord/Slack paths are available. Hermes parity, broad channel parity, voice/TTS, plugin/MCP support, and release signing are still hardening.',
  primaryCta: { label: 'Run offline in 3 commands', href: '#install' },
  secondaryCta: {
    label: 'View on GitHub',
    href: 'https://github.com/TrebuchetDynamics/gormes-agent',
  },
  proofStrip: [
    'Source build recommended',
    releaseLabel,
    'Static Go binary',
    'MIT License',
    'Offline doctor before credentials',
  ],
  installHeadline: 'Build it. Prove it offline.',
  installIntro:
    'Start with the inspectable source path. The first proof does not need credentials, a model call, Python, Docker, or Hermes. No runtime Node or npm is required.',
  installSteps: [
    {
      label: '1. BUILD FROM SOURCE',
      command:
        'git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build',
    },
    { label: '2. LOCAL DOCTOR', command: './bin/gormes doctor --offline' },
    { label: '3. OFFLINE TUI', command: './bin/gormes --offline' },
  ],
  installFootnote:
    'Provider setup, gateway setup, and convenience installers come after the offline proof.',
  installFootnoteLink: {
    label: 'Read the install docs ->',
    href: 'https://docs.gormes.ai/using-gormes/install/',
  },
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
    'Source build is the recommended scout-release path.',
    'Offline doctor runs before provider credentials or token spend.',
    'Secrets stay local under the Gormes home, not in the landing workflow.',
    'Tagged artifacts carry checksums; release signing and package-manager hardening are still in progress.',
    binaryMeasureLabel,
  ],
  builtForHeadline: 'What works today',
  builtForItems: [
    'Run a local agent UI with zero runtime dependencies on the offline path',
    'Send one-shot prompts to a provider-compatible endpoint',
    'Validate your environment before spending tokens',
    'Operate Telegram, Discord, and Slack paths from one binary when configured',
    'Inspect and debug local SQLite memory ("Goncho")',
    'Browse sessions, config, skills, and logs in the local dashboard',
  ],
  supportHeadline: 'Gateway support status',
  supportRows: [
    {
      status: 'Runtime-ready',
      body: 'Telegram, Discord, and Slack paths are promoted for configured scout-release use.',
    },
    {
      status: 'Tracked, not promoted here',
      body: 'WhatsApp, WeChat, Signal, Matrix, Mattermost, and regional channels stay in docs/roadmap status until live validation is complete.',
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
      title: 'Single Static Binary',
      body: 'CGO_ENABLED=0 release builds keep the runtime surface in one static Go binary with no Python runtime dependency.',
    },
    {
      title: 'Offline Proof',
      body: './bin/gormes --offline starts the native TUI without credentials, network calls, Python, Node, Docker, or Hermes.',
    },
    {
      title: 'Built-In Doctor',
      body: './bin/gormes doctor --offline checks local readiness before provider calls or token spend.',
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
  roadmapLabel: 'BUILD STATE',
  roadmapHeadline: 'Useful today, still early.',
  roadmapBuckets: [
    {
      title: 'Shipped in scout',
      items: [
        'Offline TUI and doctor',
        'Provider one-shots',
        'Local SQLite memory and sessions',
        'Dashboard inspection',
        'Telegram, Discord, and Slack configured paths',
      ],
    },
    {
      title: 'Hardening now',
      items: [
        'Provider routing and auth edges',
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
  progressTrackerUrl: 'https://docs.gormes.ai/building-gormes/architecture_plan/',
  exploreHeadline: 'Explore',
  exploreLinks: [
    { label: 'Quickstart', href: 'https://docs.gormes.ai/using-gormes/quickstart/' },
    { label: 'Install', href: 'https://docs.gormes.ai/using-gormes/install/' },
    { label: 'Configuration', href: 'https://docs.gormes.ai/using-gormes/configuration/' },
    { label: 'Architecture', href: 'https://docs.gormes.ai/building-gormes/architecture_plan/' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  finalCtaHeadline: 'Start offline. Add credentials later.',
  finalCtaBody:
    'The offline path proves the runtime before provider calls, gateway traffic, or token spend.',
  finalPrimaryCta: { label: 'Build from source', href: '#install' },
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

export const trackerLabel = progressTrackerLabel(progress);
