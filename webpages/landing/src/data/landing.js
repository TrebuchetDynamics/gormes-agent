import benchmarks from './benchmarks.json';
import progress from './progress.json';

const STATUS_COMPLETE = 'complete';
const STATUS_IN_PROGRESS = 'in_progress';
const STATUS_PLANNED = 'planned';

const binarySizeMB = benchmarks?.binary?.size_mb || '17';

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

function derivedPhaseStatus(phase) {
  const subphases = Object.values(phase.subphases || {});
  if (subphases.length === 0) {
    return STATUS_PLANNED;
  }

  const statuses = subphases.map(derivedSubphaseStatus);
  if (statuses.every((status) => status === STATUS_COMPLETE)) {
    return STATUS_COMPLETE;
  }
  if (statuses.some((status) => status === STATUS_COMPLETE || status === STATUS_IN_PROGRESS)) {
    return STATUS_IN_PROGRESS;
  }
  return STATUS_PLANNED;
}

function statusTone(status, phaseKey) {
  if (status === STATUS_COMPLETE) {
    return 'shipped';
  }
  if (status === STATUS_IN_PROGRESS) {
    return 'progress';
  }
  return phaseKey === '5' ? 'later' : 'planned';
}

function itemStatusMeta(status) {
  if (status === STATUS_COMPLETE) {
    return { statusLabel: 'Done', tone: 'shipped' };
  }
  if (status === STATUS_IN_PROGRESS) {
    return { statusLabel: 'Now', tone: 'ongoing' };
  }
  return { statusLabel: 'Next', tone: 'pending' };
}

function statusLabel(status, complete, total) {
  if (status === STATUS_COMPLETE) {
    return `SHIPPED · ${complete}/${total}`;
  }
  if (status === STATUS_IN_PROGRESS) {
    return `IN PROGRESS · ${complete}/${total}`;
  }
  return `PLANNED · 0/${total}`;
}

function buildItems(phase) {
  return Object.entries(phase.subphases || {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, subphase]) => {
      const status = derivedSubphaseStatus(subphase);
      return {
        ...itemStatusMeta(status),
        label: `${key} ${subphase.name}`,
      };
    });
}

export function buildRoadmapPhases(source = progress) {
  return Object.entries(source.phases || {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, phase]) => {
      const items = buildItems(phase);
      const complete = Object.values(phase.subphases || {}).filter(
        (subphase) => derivedSubphaseStatus(subphase) === STATUS_COMPLETE,
      ).length;
      const total = Object.keys(phase.subphases || {}).length;
      const phaseStatus = derivedPhaseStatus(phase);

      return {
        title: phase.name,
        statusLabel: statusLabel(phaseStatus, complete, total),
        statusTone: statusTone(phaseStatus, key),
        items,
      };
    });
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
  title: 'Gormes — AI Agents From One Go Binary',
  description:
    'Gormes runs AI agents as a single static Go binary. Start offline, prove the machine works, then add provider and gateway credentials.',
  nav: [
    { label: 'Docs', href: 'https://docs.gormes.ai/' },
    { label: 'Roadmap', href: '#roadmap' },
    { label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/gormes-agent' },
  ],
  heroKicker: 'OPEN SOURCE · MIT LICENSE · EARLY SCOUT RELEASE',
  heroHeadline: 'Run Agents From One Go Binary.',
  heroLines: [
    'Gormes runs AI agents as a single static binary.',
    'No Python runtime. No virtualenv repair. No backend service just to open the UI.',
    'Start offline, prove the machine works, then add provider and gateway credentials.',
  ],
  heroFilterStamp: 'Scout release. Useful today, still early.',
  heroFilterLine:
    'Offline TUI, doctor diagnostics, provider one-shots, Goncho memory, dashboard, and configured Telegram/Discord/Slack paths are covered. Full parity is still hardening.',
  primaryCta: { label: 'Build from source', href: '#install' },
  secondaryCta: {
    label: 'View on GitHub',
    href: 'https://github.com/TrebuchetDynamics/gormes-agent',
  },
  proofStrip: [
    `~${binarySizeMB} MB static binary`,
    'Source build recommended',
    'MIT License',
    'Scout release',
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
    { label: '2. OFFLINE TUI', command: './bin/gormes --offline' },
    { label: '3. LOCAL DOCTOR', command: './bin/gormes doctor --offline' },
  ],
  installFootnote:
    'Provider setup, gateway setup, and convenience installers come after the offline proof.',
  installFootnoteLink: {
    label: 'Read the install docs ->',
    href: 'https://docs.gormes.ai/using-gormes/install/',
  },
  builtForHeadline: 'What works today',
  builtForItems: [
    'Run a local agent UI with zero runtime dependencies on the offline path',
    'Send one-shot prompts to a provider-compatible endpoint',
    'Validate your environment before spending tokens',
    'Operate configured Telegram, Discord, or Slack agents from one binary',
    'Inspect and debug agent memory locally with Goncho',
    'Browse sessions, config, skills, and logs in the local dashboard',
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
      body: `CGO_ENABLED=0 release builds produce a ~${binarySizeMB} MB artifact for the runtime surface.`,
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
      title: 'Local Goncho Memory',
      body: 'Sessions, durable context, diagnostics, and queue state stay in local SQLite.',
    },
    {
      title: 'Visible Limits',
      body: 'Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening remain in progress.',
    },
  ],
  roadmapLabel: 'BUILD STATE',
  roadmapHeadline: 'Useful today, still early.',
  roadmapCurrentFocus: [
    'Offline TUI, doctor diagnostics, provider one-shots, dashboard, and Goncho memory',
    'Configured Telegram and Discord gateways; Slack with complete Socket Mode credentials',
    'Go-native tool registry, web/browser tools, and subagent safety',
  ],
  roadmapNextMilestone:
    'Production-stable Go-native runtime with signed releases and broader Hermes parity',
  roadmapDetailsSummary: 'View full phase-by-phase checklist',
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
};

export const roadmapPhases = buildRoadmapPhases(progress);
export const trackerLabel = progressTrackerLabel(progress);
