---
layout: default
title: Contributing
nav_order: 16
---

# Contributing Guide

Thank you for your interest in contributing to Mind Palace! This guide will help you get started.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```sh
   git clone https://github.com/YOUR_USERNAME/mind-palace.git
   cd mind-palace
   ```
3. **Set up the development environment**:
   ```sh
   make deps
   make build
   make test
   ```

See [Development Guide](./development.md) for detailed setup instructions.

## Types of Contributions

### Bug Reports

Found a bug? Please [open an issue](https://github.com/mehmetkoksal-w/mind-palace/issues/new) with:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Go version, Node version)
- Relevant logs or error messages

### Feature Requests

Have an idea? [Open an issue](https://github.com/mehmetkoksal-w/mind-palace/issues/new) describing:

- The problem you're trying to solve
- Your proposed solution
- Alternative approaches you've considered
- How this benefits other users

### Code Contributions

1. **Find or create an issue** describing what you want to work on
2. **Comment on the issue** to let others know you're working on it
3. **Create a branch** from `main`:
   ```sh
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** following the code style guidelines below
5. **Write tests** for new functionality
6. **Run the test suite**:
   ```sh
   make test
   make lint
   ```
7. **Commit your changes** with a clear message
8. **Push and open a Pull Request**

## Code Style

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting (automatic with most editors)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and reasonably sized

Example:
```go
// StartSession begins a new agent work session.
// It creates a unique session ID and initializes tracking.
func (m *Memory) StartSession(agentType, agentID, goal string) (*Session, error) {
    if agentType == "" {
        return nil, errors.New("agent type is required")
    }
    // Implementation...
}
```

### TypeScript/Angular Code

- Use TypeScript strict mode
- Follow Angular style guide
- Use meaningful component and service names
- Prefer signals over traditional observables where appropriate
- Use standalone components

Example:
```typescript
@Component({
  selector: 'app-sessions',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `...`
})
export class SessionsComponent {
  sessions = signal<Session[]>([]);

  constructor(private api: ApiService) {
    this.loadSessions();
  }

  async loadSessions() {
    const data = await this.api.getSessions();
    this.sessions.set(data);
  }
}
```

### Commit Messages

Write clear, concise commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem this commit solves and why this approach
was chosen.

- Bullet points are fine
- Use present tense ("Add feature" not "Added feature")
```

Good examples:
- `Add session memory CLI commands`
- `Fix corridor link path resolution on Windows`
- `Update dashboard to use Angular 17 signals`

Bad examples:
- `fix bug`
- `WIP`
- `changes`

## Pull Request Process

1. **Ensure all tests pass**: `make test`
2. **Ensure linting passes**: `make lint`
3. **Update documentation** if you've changed behavior
4. **Fill out the PR template** completely
5. **Link related issues** using GitHub keywords (`Fixes #123`)

### PR Title Format

Use a descriptive title:
- `feat: Add dark mode to dashboard`
- `fix: Resolve race condition in session tracking`
- `docs: Update CLI reference for new commands`
- `refactor: Simplify corridor linking logic`

### Review Process

1. A maintainer will review your PR
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR

## Testing

### Go Tests

```sh
# Run all Go tests
make test-go

# Run specific package tests
go test -v ./internal/memory/...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Dashboard Tests

```sh
# Run all tests
make test-dashboard

# Run specific test
cd apps/dashboard && npm test -- --include='**/sessions*'

# Run with coverage
cd apps/dashboard && npm test -- --code-coverage
```

### End-to-End Tests

```sh
make e2e
```

## Documentation

- Update relevant docs when adding features
- Add code examples where helpful
- Keep language clear and concise
- Test that documentation builds: The Jekyll site builds in CI

Documentation lives in `docs/` and is built with Jekyll for GitHub Pages.

## Architecture Decisions

For significant changes, consider:

1. **Discuss first** in an issue before implementing
2. **Document your approach** in the PR description
3. **Consider backwards compatibility**
4. **Think about performance implications**

## Adding New Features

### Adding CLI Commands

1. Add command in `internal/cli/cli.go`
2. Implement the handler function
3. Add tests in `internal/cli/cli_test.go`
4. Update CLI documentation in `docs/cli.md`

### Adding MCP Tools

1. Define tool schema in `internal/butler/mcp.go`
2. Implement handler function
3. Register in tool dispatcher
4. Add tests
5. Update MCP documentation

### Adding Dashboard Views

1. Create component in `apps/dashboard/src/app/features/`
2. Add route in `app.routes.ts`
3. Add navigation link
4. Implement API calls
5. Add component tests
6. Update dashboard documentation

### Adding Language Support

1. Create analyzer in `internal/analysis/`
2. Implement the `Analyzer` interface
3. Register in analyzer factory
4. Add comprehensive tests
5. Update supported languages documentation

## Community

- Be respectful and inclusive
- Help others when you can
- Give credit where due
- Follow the [Code of Conduct](https://github.com/mehmetkoksal-w/mind-palace/blob/main/CODE_OF_CONDUCT.md)

## Questions?

- Open a [discussion](https://github.com/mehmetkoksal-w/mind-palace/discussions)
- Check existing [issues](https://github.com/mehmetkoksal-w/mind-palace/issues)
- Read the [documentation](https://mehmetkoksal-w.github.io/mind-palace)

Thank you for contributing!
