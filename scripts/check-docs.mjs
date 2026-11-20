// SPDX-License-Identifier: AGPL-3.0-only
// Checks local documentation links, bilingual navigation and SVG portability.
import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import MarkdownIt from 'markdown-it';
import GithubSlugger from 'github-slugger';

const root = path.resolve(fileURLToPath(new URL('..', import.meta.url)));
const parser = new MarkdownIt({ html: true });
const errors = [];
const excluded = new Set(['.git', 'node_modules', '.artifacts']);

async function filesIn(directory) {
  const result = [];
  for (const item of await readdir(directory, { withFileTypes: true })) {
    if (excluded.has(item.name)) continue;
    const filename = path.join(directory, item.name);
    if (item.isDirectory()) result.push(...await filesIn(filename));
    else if (item.isFile()) result.push(filename);
  }
  return result;
}

const files = await filesIn(root);
const documents = new Map();
const relative = filename => path.relative(root, filename).replaceAll('\\', '/');
const fail = (filename, message) => errors.push(`${relative(filename)}: ${message}`);

for (const filename of files.filter(name => name.endsWith('.md'))) {
  const source = await readFile(filename, 'utf8');
  const tokens = parser.parse(source, {});
  const anchors = new Set();
  const links = [];
  const slugger = new GithubSlugger();
  for (let index = 0; index < tokens.length; index++) {
    const token = tokens[index];
    if (token.type === 'heading_open') {
      const inline = tokens[index + 1];
      const text = (inline?.children ?? [])
        .filter(child => ['text', 'code_inline', 'image'].includes(child.type))
        .map(child => child.content).join('');
      anchors.add(slugger.slug(text));
    }
    for (const child of token.children ?? []) {
      if (child.type === 'link_open') links.push(child.attrGet('href'));
      if (child.type === 'image') links.push(child.attrGet('src'));
    }
    if (token.type === 'html_block' || token.type === 'inline') {
      for (const match of token.content.matchAll(/\b(?:href|src)\s*=\s*["']([^"']+)["']/gi)) {
        links.push(match[1]);
      }
      for (const match of token.content.matchAll(/\bid\s*=\s*["']([^"']+)["']/gi)) {
        anchors.add(match[1]);
      }
    }
  }
  documents.set(filename, { source, anchors, links });
}

let localLinks = 0;
for (const [filename, { links }] of documents) {
  for (const link of links) {
    if (!link || /^(?:https?:|mailto:|tel:)/i.test(link)) continue;
    if (/^(?:[a-z][a-z\d+.-]*:|\/\/)/i.test(link)) {
      fail(filename, `unsupported link scheme: ${link}`);
      continue;
    }
    localLinks++;
    try {
      const hashAt = link.indexOf('#');
      const location = hashAt < 0 ? link : link.slice(0, hashAt);
      const anchor = hashAt < 0 ? '' : decodeURIComponent(link.slice(hashAt + 1));
      const target = location
        ? path.resolve(path.dirname(filename), decodeURIComponent(location.split('?')[0]))
        : filename;
      const inside = path.relative(root, target);
      if (inside.startsWith('..') || path.isAbsolute(inside)) {
        fail(filename, `link leaves the repository: ${link}`);
        continue;
      }
      if (!(await stat(target)).isFile()) {
        fail(filename, `link must name an existing file: ${link}`);
        continue;
      }
      if (anchor && documents.has(target) && !documents.get(target).anchors.has(anchor)) {
        fail(filename, `missing anchor: ${link}`);
      }
    } catch (error) {
      fail(filename, `invalid or missing local link ${link}: ${error.code ?? error.message}`);
    }
  }
}

const english = path.join(root, 'docs', 'en');
const spanish = path.join(root, 'docs', 'es');
const englishDocs = [...documents.keys()].filter(name => name.startsWith(english + path.sep));
const pairs = [
  [path.join(root, 'README.md'), path.join(root, 'README.es.md')],
  ...englishDocs.map(name => [name, path.join(spanish, path.relative(english, name))]),
];

for (const [left, right] of pairs) {
  const en = documents.get(left);
  const es = documents.get(right);
  if (!en || !es) {
    fail(en ? right : left, 'missing English/Spanish counterpart');
    continue;
  }
  const marker = /<!--\s*docs-pair:\s*([\w-]+);\s*revision:\s*(\d+)\s*-->/;
  const a = en.source.match(marker);
  const b = es.source.match(marker);
  if (!a || !b || a[1] !== b[1] || a[2] !== b[2]) {
    fail(left, 'English/Spanish revision markers differ or are missing');
  }
  for (const [filename, other] of [[left, right], [right, left]]) {
    const doc = documents.get(filename);
    const navigates = doc.links.some(link => link && !link.includes(':') &&
      path.resolve(path.dirname(filename), link.split('#')[0]) === other);
    if (!navigates) fail(filename, 'missing navigation to its language counterpart');
  }
}

for (const filename of documents.keys()) {
  if (filename.startsWith(spanish + path.sep) &&
      !documents.has(path.join(english, path.relative(spanish, filename)))) {
    fail(filename, 'Spanish document has no English counterpart');
  }
}

const banners = files.filter(name => name.endsWith('.svg'));
for (const filename of banners) {
  const source = await readFile(filename, 'utf8');
  if (/<(?:script|foreignObject)\b|\bon\w+\s*=|@import|<!DOCTYPE|<!ENTITY/i.test(source)) {
    fail(filename, 'SVG contains executable or externally extensible content');
  }
  for (const match of source.matchAll(/\b(?:href|xlink:href)\s*=\s*["']([^"']*)["']/gi)) {
    if (!match[1].startsWith('#')) fail(filename, 'SVG references an external asset');
  }
  for (const match of source.matchAll(/url\(\s*["']?([^\s)"']+)/gi)) {
    if (!match[1].startsWith('#')) fail(filename, 'SVG uses an external CSS resource');
  }
  if (!/<title\b/.test(source) || !/<desc\b/.test(source) ||
      !source.includes('prefers-reduced-motion')) {
    fail(filename, 'SVG needs a title, description and reduced-motion handling');
  }
}

if (errors.length) {
  console.error(errors.join('\n'));
  process.exitCode = 1;
} else {
  console.log(`Checked ${documents.size} Markdown files, ${localLinks} local links, ` +
    `${pairs.length} language pairs and ${banners.length} SVGs.`);
  console.log('External URLs and translation meaning require human review.');
}
