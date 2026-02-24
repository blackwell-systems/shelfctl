# Documentation

This directory contains all documentation for shelfctl.

## For Users

Start with [index.md](index.md) or jump to:
- [Tutorial](guides/tutorial.md) - Learn by doing
- [Commands](reference/commands.md) - Look up specific commands
- [Troubleshooting](reference/troubleshooting.md) - Fix problems

## For Developers

- [architecture.md](reference/architecture.md) - Schemas and configuration reference
- [contributing.md](development/contributing.md) - Development setup
- [components.md](development/components.md) - Reusable TUI components
- [tui-architecture.md](development/tui-architecture.md) - Unified TUI design

## Building Documentation Site

This structure works with multiple documentation generators:

### MkDocs Material
```bash
pip install mkdocs-material
mkdocs new .
# Edit mkdocs.yml to use docs/ folder
mkdocs serve
```

### Hugo
```bash
hugo new site .
# Copy docs/ to content/docs/
hugo server
```

### Docusaurus
```bash
npx create-docusaurus@latest my-website classic
# Copy docs/ to docs/
npm start
```

### GitHub Pages (Simple)
Just browse to `https://github.com/blackwell-systems/shelfctl/tree/main/docs` - GitHub renders markdown automatically.
