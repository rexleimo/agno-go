# HNO Documentation

This directory contains the VitePress documentation site for HNO.

## Local Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Setup

```bash
# Install dependencies
npm install

# Start dev server
npm run docs:dev

# Build for production
npm run docs:build

# Preview production build
npm run docs:preview
```

### Development Server

The dev server will be available at `http://localhost:5173` with hot reload.

## Project Structure

```
website/
├── .vitepress/
│   └── config.mjs         # VitePress configuration (ESM). If using TypeScript, use config.ts
├── index.md               # Homepage (Hero + Features)
├── guide/                 # User guides
│   ├── index.md          # What is HNO?
│   ├── quick-start.md    # 5-minute tutorial
│   ├── installation.md   # Setup instructions
│   ├── agent.md          # Agent guide
│   ├── team.md           # Team guide (placeholder)
│   ├── workflow.md       # Workflow guide (placeholder)
│   ├── models.md         # Models guide (placeholder)
│   ├── tools.md          # Tools guide (placeholder)
│   └── memory.md         # Memory guide (placeholder)
├── api/                   # API reference
│   ├── index.md          # API overview
│   ├── agent.md          # Agent API (placeholder)
│   ├── team.md           # Team API (placeholder)
│   ├── workflow.md       # Workflow API (placeholder)
│   ├── models.md         # Models API (placeholder)
│   ├── tools.md          # Tools API (placeholder)
│   ├── memory.md         # Memory API (placeholder)
│   ├── types.md          # Types API (placeholder)
│   └── agentos.md        # AgentOS API (placeholder)
├── advanced/              # Advanced topics
│   ├── architecture.md   # System architecture
│   ├── performance.md    # Performance benchmarks
│   ├── deployment.md     # Production deployment
│   └── testing.md        # Testing guide (placeholder)
└── examples/              # Code examples
    └── index.md          # Examples overview
```

## Configuration

### Site Configuration

Edit `.vitepress/config.mjs` (or `.vitepress/config.ts` if using TypeScript) to modify:

- Site title and description
- Base URL (set to `/` for the custom domain `hno.rexai.top`)
- Navigation menu
- Sidebar structure
- Theme options
- Search settings

### Important: Base URL

The custom domain is served from the site root, so `base` is `/` in `.vitepress/config.mjs`:

```ts
export default defineConfig({
  base: '/',
  // ...
})
```

## Deployment

### GitHub Pages (Automatic)

The site automatically deploys to GitHub Pages when changes are pushed to `main` branch:

1. Push changes to `main`
2. GitHub Actions builds the site
3. Deployed to `https://hno.rexai.top/`

**Workflow**: `.github/workflows/deploy-docs.yml`

### Custom Domain

The repository includes `website/public/CNAME`, so the generated Pages artifact
contains `hno.rexai.top`. GitHub and DNS still need to be configured:

1. In **Settings → Pages**, set the custom domain to `hno.rexai.top` and enable HTTPS after DNS validation.
2. At the DNS provider, create `CNAME hno -> rexleimo.github.io`.
3. Push the site changes to `main` and wait for the Pages deployment to finish.

The deployed site should then be available at `https://hno.rexai.top/` with a root
base path, not under `/HNO/`.

### Manual Deployment

```bash
# Build site
npm run docs:build

# Output in website/.vitepress/dist/
# Deploy dist/ to any static hosting
```

## Writing Documentation

### Markdown Features

VitePress supports GitHub-Flavored Markdown plus additional features:

#### Code Blocks with Syntax Highlighting

\`\`\`go
package main

func main() {
    fmt.Println("Hello, HNO!")
}
\`\`\`

#### Custom Containers

\`\`\`markdown
::: tip
This is a tip
:::

::: warning
This is a warning
:::

::: danger
This is a danger message
:::
\`\`\`

#### Code Groups

\`\`\`markdown
::: code-group
\`\`\`bash [npm]
npm install agno-go
\`\`\`

\`\`\`bash [yarn]
yarn add agno-go
\`\`\`
:::
\`\`\`

#### Line Highlighting

\`\`\`go{2,4-6}
package main

import "fmt" // highlighted

func main() { // highlighted
    fmt.Println("Hello") // highlighted
}
\`\`\`

### Internal Links

Use relative paths for internal links:

```markdown
[Quick Start](/guide/quick-start)
[API Reference](/api/)
[Agent Guide](/guide/agent)
```

## Customization

### Theme

VitePress uses Vue 3 and can be customized with Vue components.

To add custom components:

1. Create `.vitepress/theme/index.ts`
2. Import custom components
3. Register globally

### Styles

Add custom CSS in `.vitepress/theme/custom.css`.

## Troubleshooting

### Port Already in Use

Change the dev server port:

```bash
npm run docs:dev -- --port 8080
```

### Build Errors

Clear cache and rebuild:

```bash
rm -rf website/.vitepress/cache
rm -rf website/.vitepress/dist
npm run docs:build
```

### Missing Sidebar Items

Ensure your markdown files have proper frontmatter and are referenced in `.vitepress/config.mjs` (or `.ts`) sidebar configuration.

## Internationalization (i18n)

This site is configured with multiple locales using VitePress v1.x `locales`.

- Directories: create one per language, e.g. `website/zh`, `website/ja`, `website/ko`.
- Each locale should have at least `index.md`, `guide/`, `api/`, `advanced/`, `examples/` as needed.
- Add the locale to `locales` in `.vitepress/config.mjs`, with `label`, `lang`, `title`, `description`.
- Provide locale-specific nav/sidebar in that locale’s `themeConfig` (already set up in this repo).

To add a new language (e.g., Spanish `es`):

1. Create folders and seed content:
   - `website/es/index.md`
   - `website/es/guide/index.md` (and other sections as needed)
2. Update `.vitepress/config.mjs` → `locales.es = { label, lang, title, description, themeConfig }`.
3. Run `npm run docs:build` and verify `website/.vitepress/dist/es/` is generated.

## Contributing

To contribute to documentation:

1. Fork the repository
2. Create a branch: `git checkout -b docs/my-improvement`
3. Make changes in `website/` directory
4. Test locally: `npm run docs:dev`
5. Build to verify: `npm run docs:build`
6. Commit and push
7. Open a pull request

## Resources

- [VitePress Documentation](https://vitepress.dev/)
- [Markdown Guide](https://www.markdownguide.org/)
- [Vue 3 Documentation](https://vuejs.org/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)

## License

MIT License - Same as HNO project
