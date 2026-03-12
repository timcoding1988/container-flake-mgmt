# Container Flake Management

A tool to detect and track flaky tests in the `containers/podman` CI pipelines.

## Overview

This tool analyzes Cirrus CI test results from the last 30 days to identify flaky tests - tests that sometimes pass and sometimes fail on the same platform.

### Key Features

- **Platform-aware grouping**: Tests are grouped by CI matrix (e.g., `sys fedora rootless`) to avoid misclassifying platform-specific failures as flakiness
- **Concurrent fetching**: Uses worker pools to fetch artifacts in parallel with automatic retry/backoff
- **BeforeEach/AfterEach detection**: Captures setup/teardown failures, not just `[It]` blocks
- **Days since failure**: Shows how recently a test failed to help identify recently-fixed tests

## Usage

```bash
# Build
go build ./cmd/flake-detect

# Run analysis
./flake-detect \
  --repo containers/podman \
  --days 30 \
  --output docs/

# With Cirrus API token (for higher rate limits)
CIRRUS_API_TOKEN=xxx ./flake-detect --repo containers/podman
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `containers/podman` | Repository (owner/name) |
| `--branch` | `main` | Branch to analyze |
| `--days` | `30` | Number of days to analyze |
| `--output` | `docs` | Output directory for HTML report |
| `--data` | `data/results` | Directory for JSON data |
| `--workers` | `10` | Concurrent artifact fetchers |
| `--verbose` | `false` | Enable verbose logging |
| `--dry-run` | `false` | Don't write files |

## Flakiness Classification

| Classification | Failure Rate | Meaning |
|----------------|--------------|---------|
| Stable | 0% | Test always passes on this platform |
| Low | 1-10% | Occasional failures |
| Medium | 10-30% | Frequent failures |
| High | 30%+ | Very unreliable |
| Broken | 100% | Always fails (not flaky, just broken) |

## Known Limitations

- **30-day window**: Tests fixed within the window still show historical failures. Use "Days Since Fail" column to identify recently-fixed tests.
- **Artifact availability**: Some tasks may not have HTML artifacts available.

## Architecture

- `internal/cirrus/` - Cirrus CI GraphQL client with concurrent fetching
- `internal/parser/` - logformatter HTML parser (BATS/Ginkgo)
- `internal/analyzer/` - Platform-aware flakiness scoring
- `internal/reporter/` - HTML dashboard generation
- `cmd/flake-detect/` - CLI entry point

## License

Apache 2.0
