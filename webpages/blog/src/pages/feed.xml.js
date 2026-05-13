import { absoluteUrl, posts, site } from '../lib/posts.js';

export function GET() {
  const items = posts
    .map((post) => {
      const url = absoluteUrl(post.url);
      return [
        '<item>',
        `<title>${escapeXML(post.title)}</title>`,
        `<link>${url}</link>`,
        `<guid>${url}</guid>`,
        `<pubDate>${new Date(post.date).toUTCString()}</pubDate>`,
        `<description>${escapeXML(post.summary)}</description>`,
        '</item>',
      ].join('');
    })
    .join('');

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>TrebuchetDynamics Engineering</title>
    <link>${site}/</link>
    <description>Engineering notes from the validation-gated agentic porting loop behind Gormes.</description>
    <language>en-us</language>
    ${items}
  </channel>
</rss>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/rss+xml; charset=utf-8',
    },
  });
}

function escapeXML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;');
}
