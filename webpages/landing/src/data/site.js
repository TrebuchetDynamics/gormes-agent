export const site = {
  name: 'Gormes',
  runtime: 'astro-tailwind',
  url: 'https://gormes.ai/',
  docsUrl: 'https://docs.gormes.ai/',
  githubUrl: 'https://github.com/TrebuchetDynamics/gormes-agent',
  installScriptUrl: 'https://gormes.ai/install.sh',
  socialImage: 'https://gormes.ai/static/social-card.png',
  publisherName: 'TrebuchetDynamics',
  publisherUrl: 'https://trebuchetdynamics.com/',
};

export function absoluteSiteUrl(path = '/') {
  return new URL(path, site.url).toString();
}

export function absoluteDocsUrl(path = '/') {
  return new URL(path, site.docsUrl).toString();
}
