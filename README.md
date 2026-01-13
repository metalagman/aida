# aida - AI-Powered Shell Assistant

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/aida)](https://goreportcard.com/report/github.com/metalagman/aida)
[![lint](https://github.com/metalagman/aida/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/aida/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/aida/actions/workflows/test.yml)
[![version](https://img.shields.io/github/v/release/metalagman/aida?sort=semver)](https://github.com/metalagman/aida/releases)
[![license](https://img.shields.io/github/license/metalagman/aida)](LICENSE)

`aida` translates natural language prompts into shell commands and executes them directly in your terminal.

## Requirements

- Go 1.25+
- `golangci-lint` v2.8.0+ (Taskfile has `lint:install`)
- Uses Google ADK and `google.golang.org/genai` for model access and listing.
- Gemini API keys: https://aistudio.google.com/api-keys
- OpenAI API keys: https://platform.openai.com/api-keys

## Setup

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

1) Configure a provider:
```
aida providers configure aistudio
# OR
aida providers configure openai
```

2) Set the default provider (optional):
```
aida providers default openai
```

3) Set a model (optional):
```
aida providers set-model aistudio gemini-2.5-flash
# OR
aida providers set-model openai gpt-4o-mini
```

## Usage

Generate and run a command (defaults to `confirm` mode):
```
aida -- find all files in current directory and change end lines from crlf to lf
```

Execution modes:
- `--yolo`: Prints "Running: ..." and executes the command immediately.
- `--quiet`: Runs the command and displays only its output (preserves exit code).
- `--dry-run`: Prints the command but does not execute it.

Examples:
```
aida --yolo -- list files
aida --quiet -- show git status
aida --dry-run -- find large files
```

List models for a provider:
```
aida providers models aistudio
aida providers models openai --api-key YOUR_KEY
```

## Config

Config lives at:
- `~/.config/aida/config.toml`
- `~/.config/aida/config.yaml`

Example `config.toml`:
```
default_provider = "openai"
mode = "confirm"
shell = "/bin/sh"

[provider.aistudio]
api_key = "YOUR_GEMINI_KEY"
model = "gemini-2.5-flash"

[provider.openai]
api_key = "YOUR_OPENAI_KEY"
model = "gpt-4o-mini"
```

### Environment Variables

You can also configure `aida` using environment variables (which take precedence over the config file):

- `AIDA_MODE`: Execution mode (`confirm`, `yolo`, `quiet`, `dry-run`).
- `AIDA_SHELL`: Shell executable for running commands.
- `AIDA_DEFAULT_PROVIDER`: The default provider name.
- `AIDA_PROVIDER_<NAME>_API_KEY`: API key for a specific provider (e.g., `AIDA_PROVIDER_AISTUDIO_API_KEY`).
- `AIDA_PROVIDER_<NAME>_MODEL`: Model for a specific provider (e.g., `AIDA_PROVIDER_AISTUDIO_MODEL`).

## Development

```
task build
task test
task lint
```

## License

MIT License. See `LICENSE`.
