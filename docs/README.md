# Documentation

This directory contains all documentation for shelfctl.

## Structure

```
docs/
├── index.md              # Documentation home page
├── TUTORIAL.md           # Step-by-step guide for new users
├── COMMANDS.md           # Complete command reference
├── SPEC.md              # Technical specification and architecture
├── TROUBLESHOOTING.md   # Common issues and solutions
└── nav.yml              # Navigation structure for doc generators
```

## For Users

Start with [index.md](index.md) or jump to:
- [Tutorial](TUTORIAL.md) - Learn by doing
- [Commands](COMMANDS.md) - Look up specific commands
- [Troubleshooting](TROUBLESHOOTING.md) - Fix problems

## For Developers

- [SPEC.md](SPEC.md) - Architecture and design decisions
- [../CONTRIBUTING.md](../CONTRIBUTING.md) - Development setup

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
