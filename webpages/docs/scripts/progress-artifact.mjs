import path from 'node:path';

export function createProgressArtifactPlan({ docsDir, repoRoot, env = process.env } = {}) {
  if (!docsDir) {
    throw new Error('docsDir is required to create the progress artifact plan');
  }
  if (!repoRoot) {
    throw new Error('repoRoot is required to create the progress artifact plan');
  }

  const outDir = path.resolve(docsDir, env.ASTRO_OUT_DIR || 'dist');
  return {
    command: 'go',
    args: ['run', './cmd/progress', 'emit'],
    cwd: repoRoot,
    maxBuffer: 16 * 1024 * 1024,
    target: path.join(
      outDir,
      'building-gormes',
      'architecture_plan',
      'progress.json',
    ),
  };
}
