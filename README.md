# aida - AI-Powered Shell Assistant

`aida` translates natural language prompts into shell commands and executes them directly in your terminal.

## Requirements

- Go 1.25+
- `golangci-lint` v2.8.0+ (Taskfile has `lint-install`)
- Uses Google ADK and `google.golang.org/genai` for model access and listing.
- Gemini API keys: https://aistudio.google.com/api-keys

## Setup

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
aida providers set-model aistudio gemini-3-flash-preview
# OR
aida providers set-model openai --model gpt-4o
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
aida providers models openai --all
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
model = "gemini-3-flash"

[provider.openai]
api_key = "YOUR_OPENAI_KEY"
model = "gpt-4o"
```

### Environment Variables

You can also configure `aida` using environment variables (which take precedence over the config file):

- `AIDA_MODE`: Execution mode (`confirm`, `yolo`, `quiet`, `dry-run`).
- `AIDA_SHELL`: Shell executable for running commands.
- `AIDA_DEFAULT_PROVIDER`: The default provider name.
- `AIDA_PROVIDER_<NAME>_API_KEY`: API key for a specific provider (e.g., `AIDA_PROVIDER_AISTUDIO_API_KEY`).
- `AIDA_PROVIDER_<NAME>_MODEL`: Model for a specific provider (e.g., `AIDA_PROVIDER_AISTUDIO_MODEL`).
- `AIDA_LLM_API_KEY`: API key for the currently active provider.
- `AIDA_LLM_MODEL`: Model name for the currently active provider.

## Development

```
task build
task test
task lint
```

## License

MIT License. See `LICENSE`.
