# Contributing to shelfctl

Thanks for your interest in contributing to shelfctl.

## Development setup

### Prerequisites

- Go 1.21 or later
- A GitHub account with a personal access token
- Git

### Clone and build

```bash
git clone https://github.com/blackwell-systems/shelfctl
cd shelfctl
make build
```

The binary will be in `bin/shelfctl`.

### Running tests

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/catalog
```

### Testing locally

Set up a test configuration:

```bash
mkdir -p ~/.config/shelfctl
cp config.example.yml ~/.config/shelfctl/config.yml
# Edit config.yml with your test repo details
export GITHUB_TOKEN=ghp_...
```

Create a test shelf repo on GitHub (e.g., `shelf-test`) and use it for development.

## Code style

- Run `gofmt` before committing
- Follow standard Go idioms and patterns
- Add tests for new functionality
- Update documentation for user-facing changes

## Pull requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes with clear commit messages
4. Add tests if applicable
5. Run tests and ensure they pass
6. Push to your fork
7. Open a pull request against `main`

### Commit message format

Use conventional commits:

```
feat: add support for URL ingestion
fix: correct slugify double-hyphen handling
docs: update command reference
test: add catalog parsing tests
chore: update dependencies
```

### PR checklist

- [ ] Tests pass locally
- [ ] New code has tests (if applicable)
- [ ] Documentation updated (if user-facing)
- [ ] Commit messages follow conventional format
- [ ] No breaking changes (or clearly documented)

## Areas for contribution

### High priority

- Increase test coverage (especially `internal/app` package)
- Improve error messages and user feedback
- Add retry logic for GitHub API failures
- Unicode handling in slugify function
- Checksum verification during upload

### Documentation

- Additional usage examples
- Common workflow guides
- Integration examples (shell scripts, Alfred workflows, etc.)

### Features

See [architecture.md](../reference/architecture.md) for schemas and package structure.

## Questions?

Open an issue for discussion before starting work on large features or breaking changes.
