# Contributing to Undertow

Thank you for your interest in contributing to Undertow. This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | >= 1.26 | Compilation |
| Make | Any | Build orchestration |
| Git | Any | Version control |

### Getting Started

```bash
# Clone the repository
git clone https://github.com/Real-Fruit-Snacks/Undertow.git
cd Undertow

# Verify Go installation
go version

# Build all targets
./build.sh reverse 127.0.0.1:4444

# Or use Make directly
make build
```

### Project Structure

```
undertow/
├── main.go                    # Entry point, ldflags-baked variables, server wiring
├── core.go                    # Auth, port forwarding, TLS, SFTP, parameter parsing
├── out.go                     # Bracketed status output (RFS Cyclorama Dawn palette)
├── ssh_session.go             # Session dispatcher
├── ssh_session_unix.go        # Unix PTY (creack/pty)
├── ssh_session_windows.go     # Windows ConPTY
├── detach_unix.go             # Background mode (setsid)
├── detach_windows.go          # Background mode (DETACHED_PROCESS)
├── build.sh                   # Build wrapper script
├── Makefile                   # Cross-compilation targets
└── docs/                      # GitHub Pages site
```

## Code Style

### Go Standards

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for issues
- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Keep functions focused and under ~100 lines where practical

### Build Tags

Platform-specific code uses build tags:
- `//go:build !windows` for Unix code
- `//go:build windows` for Windows code

### Static Compilation

All builds must work with `CGO_ENABLED=0`. Do not introduce cgo dependencies.

## Making Changes

### Branch Naming

- `feature/description` for new features
- `fix/description` for bug fixes
- `docs/description` for documentation changes

### Commit Messages

Write clear, concise commit messages:

```
type: short description

Longer explanation if needed. Wrap at 72 characters.
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `chore`

### Pull Request Process

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Ensure `go build`, `go fmt`, and `go vet` all pass
5. Test on at least one platform (Linux or Windows)
6. Submit a pull request with a clear description

### PR Requirements

- [ ] Code compiles with `CGO_ENABLED=0`
- [ ] `gofmt -l .` produces no output
- [ ] `go vet ./...` reports no issues
- [ ] Cross-compilation works (`GOOS=linux` and `GOOS=windows`)
- [ ] Stripped binary size not materially grown (about 6.5 MB per target today)
- [ ] No cgo dependencies introduced
- [ ] Status output uses the bracketed tokens (`[ok]`, `[!]`, `[x]`, `[i]`) via `out.go` and respects `NO_COLOR`

## What We Accept

- Bug fixes
- Performance improvements
- New SSH features (subsystems, channel types)
- Platform support improvements
- TLS/evasion enhancements
- Documentation improvements
- Build system improvements

## What We Likely Won't Accept

- Changes that break static compilation
- Dependencies that require cgo
- Features that significantly increase binary size
- Changes that break backward compatibility with existing build scripts

## Testing

Since Undertow is a penetration testing tool, testing involves:

1. **Build verification:** Binary compiles for all targets
2. **Functional testing:** Reverse and bind modes work
3. **Feature testing:** SFTP, port forwarding, TLS wrapping
4. **Platform testing:** Linux and Windows behavior

```bash
# Quick build verification
./build.sh reverse 127.0.0.1:4444
ls -la bin/

# Verify binary size
du -h bin/undertow*
```

## License

By contributing to Undertow, you agree that your contributions will be licensed under the [GNU General Public License v3.0](LICENSE).
