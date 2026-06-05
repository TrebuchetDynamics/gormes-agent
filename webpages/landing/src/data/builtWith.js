import { site } from './site.js';

export const builtWithPage = {
  title: 'Built with Gormes — Real Deployments and Self-Hosted Uses',
  description:
    'A public list of real Gormes deployments and self-hosted uses, with a pull-request submission template.',
  intro:
    'Real deployments only. No fabricated customer logos, no placeholder companies. The first entry is the operator deployment that builds Gormes itself.',
  entries: [
    {
      name: 'TrebuchetDynamics operator loop',
      href: site.githubUrl,
      operator: 'TrebuchetDynamics',
      status: 'Self-hosted operator deployment',
      summary:
        'Runs the autonomous Hermes-to-Go porting loop against the public Gormes repository.',
      proof:
        'development branch progress.json, Go test gates, GitHub Actions release workflow',
      stack: ['Gormes CLI', 'progress.json', 'Go test gates', 'GitHub Actions'],
      submissionContact: `${site.githubUrl}/pulls`,
    },
  ],
  submission: {
    heading: 'Submit a deployment',
    body:
      'Open a pull request that adds one entry to webpages/landing/src/data/builtWith.js.',
    requiredFields:
      'Required fields: name, href, operator, status, summary, proof, stack, submissionContact.',
    rules: [
      'Use a real deployment, integration, or self-hosted use that can be inspected by maintainers.',
      'Describe what Gormes does in production or in the operator workflow; do not submit placeholder companies.',
      'Include proof that can be reviewed from public links, repository evidence, or a maintainer-verifiable deployment note.',
    ],
  },
};
