# Contributing to SNet

First off, thanks for taking the time to contribute! 🎉

## Code of Conduct

This project follows a simple rule: **be excellent to each other**.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/Stuko0/SNet/issues).
2. If not, open a [Bug Report](https://github.com/Stuko0/SNet/issues/new?template=bug_report.md).
3. Include details: OS, terminal emulator, Go version, nmcli version, and steps to reproduce.

### Suggesting Features

1. Open a [Feature Request](https://github.com/Stuko0/SNet/issues/new?template=feature_request.md).
2. Explain the use case and, if possible, provide a mockup or example.

### Pull Requests

1. **Fork** the repository.
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Write code** following the project conventions:
   - Go standard formatting (`gofumpt` preferred)
   - Package-level documentation on all exported symbols
   - Tests for new functionality
4. **Commit** with clear messages:
   ```bash
   git commit -m "feat: add Wi-Fi band selection to hotspot"
   ```
5. **Push** and open a PR against `main`.
6. Ensure CI passes (lint, build, test).

> The repository requires 1 approval and linear history. Commits should follow [Conventional Commits](https://www.conventionalcommits.org/).

### Development Setup

```bash
git clone https://github.com/Stuko0/SNet.git
cd SNet
go mod download
go build -o snet ./cmd/snet/
```

### Running Tests

```bash
go test ./... -v -count=1
```

### Code Style

- Run `gofmt` or `gofumpt` before committing.
- Avoid unused exports — keep the public API minimal.
- Use `internal/` packages for implementation details.
- Write table-driven tests where appropriate.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
