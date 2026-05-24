export const site = 'https://engineering.trebuchetdynamics.com';
export const siteTitle = 'TrebuchetDynamics Engineering';
export const siteDescription = 'Engineering notes from the validation-gated agentic porting loop behind Gormes.';
export const feedPath = '/feed.xml';
export const socialImagePath = '/static/go-gopher-bear-lowpoly.png';
export const githubUrl = 'https://github.com/TrebuchetDynamics/gormes-agent';

export function absoluteUrl(path = '/') {
  return new URL(path, site).toString();
}

export function articleTitle(title) {
  return `${title} - ${siteTitle}`;
}
