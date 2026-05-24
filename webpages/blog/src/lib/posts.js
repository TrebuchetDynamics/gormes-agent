const modules = import.meta.glob('../content/posts/*.md', { eager: true });

export const posts = Object.entries(modules)
  .map(([path, mod]) => {
    const slug = path.split('/').pop().replace(/\.md$/, '');
    return {
      slug,
      url: `/posts/${slug}/`,
      ...mod.frontmatter,
      Content: mod.default,
    };
  })
  .sort((left, right) => new Date(right.date) - new Date(left.date));

export { absoluteUrl } from './site.js';
