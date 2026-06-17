# gitwhy (`ghw`)

A CLI tool that wraps the GitHub CLI (`gh`) with provenance and intent tracking, built for the age of AI-generated code.

Every commit records structured metadata: who (or what) authored it, why it was made, what spec or prompt drove it, and which AI model was involved.

## Features

- **Provenance Recording** — Captures attribution, intent, spec references, and AI model info with every commit
- **Model Auto-Detection** — Automatically detects the active AI model from environment variables at commit time
- **GitHub Integration** — Wraps `gh` CLI seamlessly; all GitHub commands still work
- **Storage Backends** — Pluggable backends (git-notes or file-based storage)
- **Drift Detection** — Detects when tracked spec files have changed after code generation
- **Audit & Export** — Query and export provenance records for compliance and analysis

## Installation

### From Source

**Requirements:**
- Go 1.26.4 or later
- Git
- GitHub CLI (`gh`) installed and configured

**Build and install:**

```bash
git clone https://github.com/anomalyco/gitwhy.git
cd gitwhy
make install
```

This installs `ghw` to your `$GOBIN` (usually `~/go/bin`). Add it to your `$PATH` if needed:

```bash
export PATH="$HOME/go/bin:$PATH"
```

If `make install` does not produce a binary (known issue on some Go versions), build directly to a directory in your PATH:

```bash
go build -o /opt/homebrew/bin/ghw .   # macOS Homebrew
# or
go build -o /usr/local/bin/ghw .       # Linux/macOS
```

### Verify Installation

```bash
ghw --version
ghw --help
```

## Quick Start

### Initialize gitwhy in a repository

```bash
cd your-repo
ghw init
```

This creates a `.gitwhy/` config directory.

### Commit with provenance

```bash
ghw commit --message "Add feature" --by copilot --intent "Implement login flow" --model "claude-sonnet-4"
```

Flags are optional; `ghw commit` auto-detects the AI model from environment variables.

### View commit provenance

```bash
ghw why <commit-hash>
```

### List commits with intent annotations

```bash
ghw log
```

### Audit provenance records

```bash
ghw audit export      # Export all provenance records
ghw audit summary     # Summarize provenance data
```

## Configuration

Configuration lives in `.gitwhy/config.yaml`:

```yaml
storage:
  backend: git-notes  # or 'file'
summary:
  enabled: true
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
