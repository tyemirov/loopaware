# Contributing to LoopAware

Thank you for your interest in contributing to LoopAware. This guide covers how to get involved.

## License

LoopAware is source-available software. By submitting a contribution, you agree that your contribution will be licensed under the same terms as the project (see [LICENSE](./LICENSE)). You also grant Marco Polo Research Lab a perpetual, irrevocable, worldwide license to use, modify, and relicense your contribution as part of the Software.

## Getting Started

1. Fork the repository and clone your fork.
2. Run the development stack with Docker Compose:

```bash
./scripts/up.sh
```

3. Verify the setup:

```bash
make test
```

## Development Workflow

All code must pass the following checks before merge:

```bash
make format   # Auto-format Go and JS
make lint      # Lint Go and JS
make test      # Run full test suite
```

The project uses an inverted test pyramid: integration tests (Playwright) are the primary verification layer. Unit tests supplement where integration coverage is impractical.

## Submitting Changes

1. Create a branch from `master` for your change.
2. Write or update tests for your change.
3. Run `make format && make lint && make test` and confirm all checks pass.
4. Open a pull request with a clear description of the change and its motivation.

## What to Contribute

Good first contributions include:

- Bug reports with reproduction steps
- Documentation improvements
- Test coverage improvements
- Bug fixes with corresponding tests

For feature work, open an issue first to discuss the approach before investing time in implementation.

## Code Style

- **Go**: Follow standard `gofmt` conventions. No single-letter variables.
- **JavaScript**: Vanilla JS, no build step. All third-party dependencies must load from pinned CDN URLs.
- **HTML/CSS**: Bootstrap 5 for layout. No vendored assets.

## Reporting Issues

Open a GitHub issue with:

- A clear title describing the problem
- Steps to reproduce
- Expected behavior vs. actual behavior
- Environment details (OS, browser, Go version)

## Contact

For questions about contributing, open a GitHub issue or reach out at legal@mprlab.com.
