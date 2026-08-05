import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const siteRoot = path.join(root, 'website')
const outputPath = path.join(siteRoot, 'public', 'rss.xml')
const siteUrl = 'https://hno.rexai.top'

const sources = [
  { dir: path.join(siteRoot, 'blog'), prefix: '/blog/', language: 'en' },
  { dir: path.join(siteRoot, 'zh', 'blog'), prefix: '/zh/blog/', language: 'zh-CN' },
]

function escapeXml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
}

function frontMatter(markdown) {
  const match = markdown.match(/^---\r?\n([\s\S]*?)\r?\n---/)
  if (!match) return null
  const block = match[1]
  const value = (key) => {
    const line = block.match(new RegExp(`^${key}:\\s*["']?([^"'\\r\\n]+)["']?\\s*$`, 'm'))
    return line?.[1]?.trim() || ''
  }
  const tags = block.match(/^tags:\s*\r?\n((?:\s+-\s+.*\r?\n?)+)/m)?.[1]
    ?.split(/\r?\n/)
    .map((line) => line.replace(/^\s+-\s+/, '').trim())
    .filter(Boolean) || []
  return {
    title: value('title'),
    description: value('description'),
    date: value('date'),
    category: value('category'),
    tags,
  }
}

const items = []
for (const source of sources) {
  if (!fs.existsSync(source.dir)) continue
  for (const name of fs.readdirSync(source.dir)) {
    if (!name.endsWith('.md') || name === 'index.md') continue
    const filePath = path.join(source.dir, name)
    const metadata = frontMatter(fs.readFileSync(filePath, 'utf8'))
    if (!metadata?.title || !metadata.date) continue
    const slug = name.slice(0, -3)
    items.push({
      ...metadata,
      language: source.language,
      url: `${siteUrl}${source.prefix}${slug}`,
    })
  }
}

items.sort((a, b) => b.date.localeCompare(a.date) || a.title.localeCompare(b.title))
const latest = items.slice(0, 30)
const itemXml = latest.map((item) => {
  const pubDate = new Date(`${item.date}T00:00:00Z`).toUTCString()
  const categories = [...new Set([item.category, ...item.tags].filter(Boolean))]
    .map((category) => `    <category>${escapeXml(category)}</category>`)
    .join('\n')
  return [
    '  <item>',
    `    <title>${escapeXml(item.title)}</title>`,
    `    <link>${escapeXml(item.url)}</link>`,
    `    <guid isPermaLink="true">${escapeXml(item.url)}</guid>`,
    `    <pubDate>${pubDate}</pubDate>`,
    `    <dc:language>${escapeXml(item.language)}</dc:language>`,
    `    <description>${escapeXml(item.description)}</description>`,
    categories,
    '  </item>',
  ].filter(Boolean).join('\n')
}).join('\n')

const rss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>HNO Blog</title>
    <link>${siteUrl}/</link>
    <description>Original engineering notes and timely analysis for Go-native AI agent systems.</description>
    <language>en</language>
    <atom:link href="${siteUrl}/rss.xml" rel="self" type="application/rss+xml" />
${itemXml}
  </channel>
</rss>
`

fs.mkdirSync(path.dirname(outputPath), { recursive: true })
fs.writeFileSync(outputPath, rss)
console.log(`Generated ${latest.length} blog RSS items at ${path.relative(root, outputPath)}`)
