# Contributing

This document covers local development, verification, and maintainer release workflows for `aida`.

For installation and end-user usage, see [README.md](./README.md).

## Requirements

- Go 1.25+
- `golangci-lint` v2.8.0+ (`task lint:install`)
- `Taskfile.yml`

The project uses Google ADK and `google.golang.org/genai` under the hood, but contributors normally interact through the Go toolchain and task targets.

## Development

Build from source:

```bash
task build
```

Install into `GOPATH/bin`:

```bash
task install:go
```

Run locally:

```bash
task run -- --help
```

## Verification

Run lint:

```bash
task lint
```

Run unit and race tests:

```bash
task test
```

Provider-specific ACP integration tests are opt-in and depend on local CLI/auth state:

```bash
task test:integration
task test:integration:opencode
task test:integration:gemini
```

Integration tags:

- Codex: `integration && codex`
- OpenCode: `integration && opencode`
- Gemini: `integration && gemini`

## Release And Packaging

GitHub binary releases are built by [release.yml](./.github/workflows/release.yml).

NPM publishing is handled separately by [omnidist-release.yml](./.github/workflows/omnidist-release.yml), using `.omnidist/omnidist.yaml` as the authoritative Omnidist config. Maintainers need `NPM_PUBLISH_TOKEN` configured in GitHub Actions secrets.

Useful Omnidist tasks:

```bash
task omnidist:build
task omnidist:stage
task omnidist:verify
```

Release verification should include lint and tests before building or publishing artifacts.
