# aida - AI Shell Wrapper

Oneshot runner for LLM-generated shell commands.

## Requirements

- Go 1.25+
- `golangci-lint` (Taskfile has `lint-install`)
- Uses Google ADK and `google.golang.org/genai` for model access and listing.
- Google ADK: https://github.com/google/adk-go
- Gemini API keys: https://aistudio.google.com/api-keys

## Setup

1) Configure a provider:
```
aida providers configure aistudio
```

2) Set a model (optional):
```
aida providers set-model aistudio gemini-3-flash-preview
```

## Usage

Generate and run a command:
```
aida -- find all files in current directory and change end lines from crlf to lf
```

Execution modes (`--mode` or `-M`):
- `confirm` (default): Prompts for confirmation before running.
- `yolo`: Prints "Running: ..." and executes the command immediately.
- `quiet`: Runs the command with no output.
- `print`: Prints the command but does not execute it.

Examples:
```
aida -M yolo -- list files
aida -M quiet -- show git status
aida -M print -- find large files
```

List models for a provider:
```
aida providers models aistudio
aida providers models aistudio --all
```

## Config

Config lives at:
- `~/.config/aida/config.toml`
- `~/.config/aida/config.yaml`

Example `config.toml`:
```
default_provider = "aistudio"
mode = "confirm"
shell = "/bin/sh"

[provider.aistudio]
api_key = "YOUR_KEY"
model = "gemini-3-flash"
```

### Environment Variables

You can also configure `aida` using environment variables (which take precedence over the config file):

- `AIDA_MODE`: Execution mode (`confirm`, `yolo`, `quiet`, `print`).
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
