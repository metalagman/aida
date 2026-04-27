# aida - AI-Powered Shell Assistant

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/aida)](https://goreportcard.com/report/github.com/metalagman/aida)
[![lint](https://github.com/metalagman/aida/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/aida/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/test.yml)
[![version](https://img.shields.io/github/v/release/metalagman/aida?sort=semver)](https://github.com/metalagman/aida/releases)
[![license](https://img.shields.io/github/license/metalagman/aida)](LICENSE)

![aida logo](./docs/assets/logo.png)

`aida` (айда) translates natural language prompts into shell commands and executes them directly in your terminal.

## Requirements

- Go 1.25+
- `golangci-lint` v2.8.0+ (Taskfile has `lint:install`)
- Uses Google ADK and `google.golang.org/genai`.
- ACP-compatible CLIs are the preferred runtime path.
- Gemini API keys: https://aistudio.google.com/api-keys
- OpenAI API keys: https://platform.openai.com/api-keys

## Installation

Download the binary from the latest GitHub release:

```
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-linux-amd64
chmod +x /usr/local/bin/aida
```

macOS (Apple Silicon):
```
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-darwin-arm64
chmod +x /usr/local/bin/aida
```

Linux (arm64):
```
curl -L -o /usr/local/bin/aida https://github.com/metalagman/aida/releases/latest/download/aida-linux-arm64
chmod +x /usr/local/bin/aida
```

Replace the URL with the appropriate artifact from:
https://github.com/metalagman/aida/releases/latest

Initialize the canonical config file:
```
aida init
```

This writes `~/.config/aida/config.yaml` with detected ACP providers in the live config and a full commented reference block for ACP, pool, and API-backed provider shapes.

If no ACP CLI is detected in `PATH`, set `aida.provider` manually after init.

Example:
```
aida init --force
```

## Usage

Generate and run a command (defaults to `confirm` mode):
```
aida -- find all files in current directory and change end lines from crlf to lf
```

Execution modes:
- `confirm` (Default): Displays the generated command and prompts for confirmation before execution.
- `--yolo`: Prints "Running: <command>..." and executes it immediately without prompting.
- `--quiet`: Runs the command silently, displaying only the command's own output.
- `--dry-run`: Outputs the generated command to the terminal without executing it.

Examples:
```
aida --yolo -- list files
aida --quiet -- show git status
aida --dry-run -- find large files
aida --profile api -- list the largest files in this repo
```

## Config

Config lives at:
- `~/.config/aida/config.yaml`

Canonical reference shape:
```
runtime:
  providers:
    codex:
      type: codex_acp
      codex_acp:
        model: gpt-5.3-codex
    opencode:
      type: opencode_acp
      opencode_acp:
        model: opencode/big-pickle
        mode: plan
    copilot:
      type: copilot_acp
      copilot_acp:
        model: gpt-5-codex
    gemini:
      type: gemini_acp
      gemini_acp:
        model: gemini-3-flash-preview
        mode: plan
    claude_code:
      type: claude_code_acp
      claude_code_acp:
        model: claude-sonnet-4
    custom:
      type: generic_acp
      generic_acp:
        cmd: [custom-acp, --stdio]
        extra_args: []
        model: custom-model
        mode: ""
    openai:
      type: openai
      openai:
        api_key: ""
        model: gpt-4o-mini
    aistudio:
      type: aistudio
      aistudio:
        api_key: ""
        model: gemini-2.5-flash
    pool:
      type: pool
      pool:
        members: [codex, openai]
  mcp_servers: {}
aida:
  provider: codex
  mode: confirm
  shell: /bin/sh
profiles:
  default:
    aida:
      provider: codex
  api:
    aida:
      provider: openai
      mode: confirm
      shell: /bin/sh
```

By default, `aida init` writes only detected ACP providers into the live `runtime.providers` section. The `openai`, `aistudio`, `pool`, and example profile entries above are kept in the commented reference block for manual use.

### Environment Variables

Environment values override the YAML file:

- `AIDA_PROFILE`: Selects a profile before command execution.
- `AIDA_PROVIDER`: Overrides `aida.provider`.
- `AIDA_MODE`: Execution mode (`confirm`, `yolo`, `quiet`, `dry-run`).
- `AIDA_SHELL`: Shell executable for running commands.
- Any existing `aida.*` or `runtime.*` leaf can also be overridden by uppercasing the path and replacing dots with underscores.
- Example: `AIDA_RUNTIME_PROVIDERS_OPENAI_OPENAI_API_KEY`
- Example: `AIDA_RUNTIME_PROVIDERS_AISTUDIO_AISTUDIO_MODEL`

## Development

```
task build
task test
task test:integration
task test:integration:opencode
task test:integration:gemini
task lint
```

`task test:integration` runs the Codex ACP integration test behind `integration && codex`. `task test:integration:opencode` runs OpenCode behind `integration && opencode`. `task test:integration:gemini` runs Gemini behind `integration && gemini`. These are intentionally separate from `go test ./...` because they depend on local CLI/auth state.

## License

MIT License. See `LICENSE`.
