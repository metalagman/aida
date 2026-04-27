# Requirements
- Project: oneshot runner for LLM-generated shell commands.
- Default behavior: run commands in `confirm` mode (prompts for confirmation).
- Execution Flags:
    - `--yolo`: Standard execution with "Running: ..." output (no prompt).
    - `--quiet`: Run silently with no output.
    - `--dry-run`: Dry-run; output the command only.
- Usage example: `aida --yolo -- find files and change crlf to lf`.
- Config location: `~/.config/aida/config.yaml`.
- Config format: relay-style YAML with top-level `runtime`, `aida`, and `profiles`.
- Provider runtime: ACP agents are top priority; local API-backed provider types `openai` and `aistudio` are supported through the shared `github.com/normahq/runtime` provider registry.
- Bootstrap command: `aida init` generates a live ACP-only config from detected ACP providers plus a full commented canonical reference block.
- Provider management: do not add `aida providers` commands; config is managed by `aida init` and direct YAML edits.
- Profile selection: support `--profile` and `AIDA_PROFILE`.
- Runtime selection: support `--provider` as a one-shot provider ID override.
- Runtime overrides: support `--model` and `--api-key` as per-invocation overrides.
- Tech stack: Go + Google ADK (google.golang.org/adk + google.golang.org/genai).
- CLI/config libs: use Cobra + Viper.
- Linting: `golangci-lint` v2.8.0+ is required.
- Tests: tests must be written in `*_test` packages.
- ACP integration tests: use provider-specific build tags only. Run Codex via `integration && codex`, OpenCode via `integration && opencode`, and Gemini via `integration && gemini`.
- Build tasks: use `Taskfile`.
- Development: always write code as a senior Go developer.
- Go style: follow Go Google style decisions, Go Google best practices, and the Go Google style guide.
- Commits: follow Conventional Commits.
- Packaging: use `internal/` for non-exported code; only public APIs stay outside `internal/`.
- Release workflow: run lint and tests before building release artifacts.

## Development workflow
1. Plan the work.
2. For new features, update AGENTS.md with new requirements before implementation.
3. Draft implementation: minimal code to satisfy the requirement.
4. Update AGENTS.md with latest changes to envs, args, and related behavior.
5. Lint & static analysis: `go tool golangci-lint run` (or project task).
6. Verify with tests: `go test ./...` and `go test -race ./...`. Run the provider-specific ACP integration tasks when local auth is available.
7. Refactor & optimize: clean up to senior standards.
8. Repeat steps 3-7 as needed until clean.
9. Final verification: re-run lint and tests to ensure no regressions.
10. Update README.md with latest changes to envs, args, and related behavior.
