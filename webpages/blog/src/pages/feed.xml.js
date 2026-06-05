import { posts } from '../lib/posts.js';
import { absoluteUrl, feedPath, site, siteDescription, siteTitle } from '../lib/site.js';

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
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>${escapeXML(siteTitle)}</title>
    <link>${site}/</link>
    <description>${escapeXML(siteDescription)}</description>
    <language>en-us</language>
    <atom:link href="${absoluteUrl(feedPath)}" rel="self" type="application/rss+xml"/>
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
