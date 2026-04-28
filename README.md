# aida

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/aida)](https://goreportcard.com/report/github.com/metalagman/aida)
[![lint](https://github.com/metalagman/aida/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/aida/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/test.yml)
[![version](https://img.shields.io/github/v/release/metalagman/aida?sort=semver)](https://github.com/metalagman/aida/releases)
[![license](https://img.shields.io/github/license/metalagman/aida)](LICENSE)

![aida logo](./docs/assets/logo.png)

Turn popular CLI agents into a one-shot shell command runner.

`aida` (айда) works with `Codex`, `OpenCode`, `Gemini`, `Copilot`, and `Claude Code` to generate terminal commands, then runs them with `confirm` mode by default so you stay in control.

## Quickstart

```bash
npm install -g @metalagman/aida
aida init
aida -- list the largest files in this repo
```

`aida init` writes `~/.config/aida/config.yaml` with detected ACP providers in the live config and a full commented reference block for ACP, pool, and API-backed provider shapes.

If `aida init` does not detect a supported ACP runtime in `PATH`, edit `~/.config/aida/config.yaml` before your first command and choose a provider through `aida.provider` or a profile.

## Runtime Options

ACP-compatible CLIs are the preferred runtime path. If `codex`, `opencode`, `gemini`, `copilot`, `claudecode`, or `claude` is already available in `PATH`, `aida init` will detect it and write a working provider entry for you.

If you want to use an API-backed provider instead, edit `~/.config/aida/config.yaml` after `aida init` and configure one of these provider types:

- `openai`
- `aistudio`

API key setup:

- Gemini API keys: https://aistudio.google.com/api-keys
- OpenAI API keys: https://platform.openai.com/api-keys

## Installation

Install from npm:

```bash
npm install -g @metalagman/aida
```

Or install a release binary:

```bash
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-linux-amd64
chmod +x /usr/local/bin/aida
```

macOS (Apple Silicon):

```bash
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-darwin-arm64
chmod +x /usr/local/bin/aida
```

Linux (arm64):

```bash
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-linux-arm64
chmod +x /usr/local/bin/aida
```

See the latest release artifacts at:
https://github.com/metalagman/aida/releases/latest

## Usage

Generate and run a command:

```bash
aida -- find all files in current directory and change end lines from crlf to lf
```

Execution modes:

- `confirm` (default): show the generated command and ask before running it
- `--yolo`: print `Running: ...` and execute immediately
- `--quiet`: execute silently
- `--dry-run`: print the generated command without executing it

Examples:

```bash
aida --yolo -- list files
aida --quiet -- show git status
aida --dry-run -- find large files
aida --profile work -- list the largest files in this repo
aida --profile work --model gpt-4o-mini -- list merged branches
```

Useful flags:

- `--profile`: select a named profile from config
- `--model`: override the selected provider model for one invocation
- `--shell`: override the shell used to execute the generated command

## Config

Config lives at `~/.config/aida/config.yaml`.

`aida init` is the normal starting point:

```bash
aida init
```

To rewrite an existing config:

```bash
aida init --force
```

The config format is relay-style YAML with top-level `runtime`, `aida`, and `profiles`.

By default, `aida init` writes only detected ACP providers into the live `runtime.providers` section. The generated file also includes a commented reference block for ACP, pool, and API-backed provider shapes that you can copy from when editing the config manually.

Minimal ACP example:

```yaml
runtime:
  providers:
    codex:
      type: codex_acp
      codex_acp:
        model: gpt-5.3-codex
aida:
  provider: codex
  mode: confirm
  shell: /bin/sh
```

Minimal API-backed example:

```yaml
runtime:
  providers:
    openai:
      type: openai
      openai:
        api_key: ${OPENAI_API_KEY}
        model: gpt-4o-mini
aida:
  provider: openai
  mode: confirm
  shell: /bin/sh
```

You can also put a literal API key in YAML, but environment expansion is the better default for local secrets.

Profile example:

```yaml
profiles:
  work:
    aida:
      provider: openai
```

Environment values can also override the YAML file. The most useful ones are:

- `AIDA_PROFILE`
- `AIDA_MODE`
- `AIDA_SHELL`

## Contributing

Source builds, tests, linting, integration test tags, and release workflow notes live in [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT License. See `LICENSE`.
