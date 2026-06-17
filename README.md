# gitwhy (`ghw`)

A CLI tool that wraps the GitHub CLI (`gh`) with provenance and intent tracking, built for the age of AI-generated code.

Every commit records structured metadata: who (or what) authored it, why it was made, what spec or prompt drove it, and which AI model was involved.

## Commands

| Command | Description |
|---------|-------------|
| `ghw commit` | Commit with provenance recording (auto-captures model, ticket, intent) |
| `ghw init` | Initialize gitwhy in a repository (installs post-commit hook) |
| `ghw why <ref>` | Show full provenance record for a commit |
| `ghw log [--why]` | Show commit log with optional intent annotations |
| `ghw diff` | Show diff with optional drift detection |
| `ghw config get/set` | View or modify `.gitwhy/config.yaml` |
| `ghw pr create/view/list` | PR operations with provenance flags |
| `ghw audit export/summary` | Export or summarize provenance records |

## Auto-Capture Features (zero-friction)

These run automatically on `git commit` via the post-commit hook:

| Feature | What it captures | Source |
|---------|-----------------|--------|
| **Attribution** | Who authored the change | `--by` flag or `auto_capture.default_by` in config |
| **Model** | AI model active at commit time | `$ANTHROPIC_MODEL`, `$OPENAI_MODEL`, `$CLAUDE_MODEL`, `$GITHUB_MODEL`, `$AI_MODEL` |
| **Ticket** | Ticket/issue reference from branch name | Parsed from branch via `[A-Z]+-\d+` regex |
| **Intent** | One-line summary of changes | LLM summary (default `llm` CLI) or conventional commit parsing |
| **Branch** | Git branch at commit time | `git rev-parse --abbrev-ref HEAD` |
| **Commit hash** | Target commit | `git rev-parse HEAD` |

## Explicit Flags

Override or supplement auto-capture with `ghw commit`:

| Flag | Description |
|------|-------------|
| `--by` | Attribution: `human`, `copilot`, `agent:<name>` |
| `--intent` | Description of why the change was made |
| `--origin` | Origin type: `human`, `spec`, `prompt`, `template`, `upstream` |
| `--ticket` | Ticket or issue reference |
| `--spec` | Reference to the spec driving the change |
| `--spec-hash` | Hash of spec content at generation time |
| `--prompt` | Prompt used (if AI-generated) |
| `--model` | Model name (overrides auto-detection) |
| `-m / --message` | Commit message |

## Configuration

```yaml
# .gitwhy/config.yaml
backend: git-notes           # storage backend: git-notes | file
auto_capture:
  enabled: true
  default_by: agent:opencode  # default attribution for auto-capture
summary:
  enabled: true               # LLM-generated intent summary
  command: llm                # any CLI accepting a prompt as last arg
  mode: filenames              # filenames | diff
```

## Installation

### From Source

**Requirements:**
- Go 1.26.4 or later
- Git
- GitHub CLI (`gh`) installed and configured

**Build and install:**

```bash
git clone https://github.com/surajsrivastav/gitwhy.git
cd gitwhy
make build
```

Then copy the `ghw` binary to a directory in your `$PATH`:

```bash
# macOS (Homebrew)
cp ghw /opt/homebrew/bin/

# Linux / macOS
sudo cp ghw /usr/local/bin/

# Any (add to PATH)
mkdir -p ~/bin && cp ghw ~/bin && export PATH="$HOME/bin:$PATH"
```

Verify:

```bash
ghw version
```

### Verify Installation

```bash
ghw --version
ghw --help
```

## Quick Start

```bash
cd your-repo
ghw init                           # one-time setup
git add . && git commit -m "feat: add login"    # commits as usual
ghw why HEAD                       # see provenance
```

## Project Structure

```
cmd/          - Cobra command implementations
pkg/
  provenance/ - Provenance data model and schema
  config/     - Configuration management
  storage/    - Pluggable storage backends
  drift/      - Spec drift detection
  audit/      - Export and summary generation
  passthrough/ - GitHub CLI passthrough
```

## Testing

```bash
make test          # Run all tests
make coverage      # Generate coverage report
make vet           # Run go vet
make lint          # Run golangci-lint (if installed)
```

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or pull request on GitHub.

## Questions?

Refer to the [PRD](PRD.md) for detailed feature documentation and [decisions/](decisions/) for architecture decisions.
