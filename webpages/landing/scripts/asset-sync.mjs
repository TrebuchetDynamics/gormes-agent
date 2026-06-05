import { resolve } from 'node:path';

export function createLandingAssetCopyPlan({ repoRoot, siteRoot } = {}) {
  if (!repoRoot) {
    throw new Error('repoRoot is required to create the landing asset copy plan');
  }
  if (!siteRoot) {
    throw new Error('siteRoot is required to create the landing asset copy plan');
  }

  return {
    copies: [
      copy('install.sh', resolve(repoRoot, 'install.sh'), resolve(siteRoot, 'public/install.sh')),
      copy('install.ps1', resolve(repoRoot, 'scripts/install.ps1'), resolve(siteRoot, 'public/install.ps1')),
      copy('install.cmd', resolve(repoRoot, 'scripts/install.cmd'), resolve(siteRoot, 'public/install.cmd')),
      copy('benchmarks.json', resolve(repoRoot, 'benchmarks.json'), resolve(siteRoot, 'src/data/benchmarks.json')),
      // webpages/landing/src/data/progress.json removed 2026-05-16
      // (backlog-efficiency #1): nothing in the Astro site imports it — it was a
      // dead 5.2 MB verbatim mirror of the canonical backlog. The active Astro
      // landing page no longer consumes the retired Go renderer progress embed.
      copy(
        'gormes-agent-logo-blue.svg',
        resolve(repoRoot, 'assets/gormes-agent-logo-blue.svg'),
        resolve(siteRoot, 'public/static/gormes-agent-logo-blue.svg'),
      ),
    ],
  };
}

export function planBenchmarkRefresh({ binaryExists, forceRefresh = false, gitStatus } = {}) {
  if (!binaryExists) {
    return {
      action: 'skip',
      message: 'benchmark refresh skipped: bin/gormes is not built',
    };
  }
  if (!forceRefresh && gitStatus?.status === 0 && gitStatus.stdout.trim() !== '') {
    return {
      action: 'skip',
      message: 'benchmark refresh skipped: worktree has local changes',
    };
  }
  return { action: 'record' };
}

export function parseReleaseData(raw, { source = 'cmd/gormes/main.go', errorSource = source } = {}) {
  const match = raw.match(/var\s+Version\s*=\s*"([^"]+)"/);
  if (!match) {
    throw new Error(`could not read Version from ${errorSource}`);
  }
  const aliasMatch = raw.match(/var\s+VersionDateAlias\s*=\s*"([^"]+)"/);
  if (!aliasMatch) {
    throw new Error(`could not read VersionDateAlias from ${errorSource}`);
  }

  const version = match[1];
  const dateAlias = aliasMatch[1];
  return {
    version,
    tag: `v${version}`,
    date_alias: dateAlias,
    url: `https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v${version}`,
    source,
  };
}

function copy(label, source, target) {
  return { label, source, target };
}
