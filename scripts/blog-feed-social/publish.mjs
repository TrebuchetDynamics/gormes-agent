#!/usr/bin/env node
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

export function parseFeed(xml) {
  const channel = firstMatch(xml, /<channel>([\s\S]*?)<\/channel>/i) ?? xml;
  const title = decodeXML(firstMatch(channel, /<title>([\s\S]*?)<\/title>/i) ?? '');
  const items = [...channel.matchAll(/<item>([\s\S]*?)<\/item>/gi)].map((match) => {
    const item = match[1];
    return {
      title: decodeXML(firstMatch(item, /<title>([\s\S]*?)<\/title>/i) ?? ''),
      link: decodeXML(firstMatch(item, /<link>([\s\S]*?)<\/link>/i) ?? ''),
      guid: decodeXML(firstMatch(item, /<guid>([\s\S]*?)<\/guid>/i) ?? ''),
      pubDate: decodeXML(firstMatch(item, /<pubDate>([\s\S]*?)<\/pubDate>/i) ?? ''),
      description: decodeXML(firstMatch(item, /<description>([\s\S]*?)<\/description>/i) ?? ''),
    };
  });
  return { title, items };
}

export function buildDryRunPost(feed, options = {}) {
  if (!feed?.items?.length) {
    throw new Error('feed has no publishable items');
  }
  const item = feed.items[0];
  const canonicalURL = item.link || item.guid;
  if (!item.title || !canonicalURL) {
    throw new Error('feed item must include title and link or guid');
  }
  const idempotencyBasis = item.guid || canonicalURL;
  const idempotencyHash = createHash('sha256').update(idempotencyBasis).digest('hex').slice(0, 16);
  const siteTitle = feed.title || 'TrebuchetDynamics Engineering';
  return {
    mode: 'dry-run',
    platform: options.platform || 'operator-selected-social',
    source_feed: options.feedPath || '',
    title: item.title,
    summary: item.description,
    canonical_url: canonicalURL,
    idempotency_key: `td-blog:${idempotencyHash}`,
    post_text: `New ${siteTitle} post: ${item.title}\n${canonicalURL}`,
    network_publish: false,
    secret_policy: 'no social tokens are read in dry-run mode',
  };
}

export function run(argv = process.argv.slice(2)) {
  const args = parseArgs(argv);
  if (!args.dryRun) {
    throw new Error('live publishing is not implemented; rerun with --dry-run');
  }
  if (!args.feed) {
    throw new Error('missing required --feed <path>');
  }
  if (!args.out) {
    throw new Error('missing required --out <path>');
  }

  const feed = parseFeed(readFileSync(args.feed, 'utf8'));
  const post = buildDryRunPost(feed, { feedPath: args.feed, platform: args.platform });
  writeFileSync(args.out, `${JSON.stringify(post, null, 2)}\n`);
  return `dry-run social preview written: ${args.out}\nidempotency_key: ${post.idempotency_key}\n`;
}

function parseArgs(argv) {
  const args = { dryRun: false };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--dry-run':
        args.dryRun = true;
        break;
      case '--feed':
        args.feed = valueAfter(argv, ++i, arg);
        break;
      case '--out':
        args.out = valueAfter(argv, ++i, arg);
        break;
      case '--platform':
        args.platform = valueAfter(argv, ++i, arg);
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  return args;
}

function valueAfter(argv, index, flag) {
  const value = argv[index];
  if (!value || value.startsWith('--')) {
    throw new Error(`missing value for ${flag}`);
  }
  return value;
}

function firstMatch(value, pattern) {
  return pattern.exec(value)?.[1]?.trim();
}

function decodeXML(value) {
  return String(value)
    .replaceAll('&apos;', "'")
    .replaceAll('&quot;', '"')
    .replaceAll('&gt;', '>')
    .replaceAll('&lt;', '<')
    .replaceAll('&amp;', '&');
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    process.stdout.write(run());
  } catch (error) {
    process.stderr.write(`error: ${error.message}\n`);
    process.exitCode = 1;
  }
}
