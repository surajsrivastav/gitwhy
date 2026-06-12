# ADR 001: Project Structure and Language Choice

**Date:** 2026-06-05
**Status:** Accepted

## Context
gitwhy needs a CLI tool (`ghw`) that wraps GitHub CLI (`gh`) with provenance tracking. The PRD specifies Go as the implementation language.

## Decision
### Language: Go
- Matches `gh` architecture (single binary, fast startup)
- Excellent CLI library ecosystem (Cobra, Viper)
- Cross-platform compilation target (darwin, linux, windows)

### CLI Framework: Cobra
- De facto standard for Go CLIs (used by `gh`, `kubectl`, `docker`, etc.)
- Built-in help generation, flag parsing, subcommand routing
- `cobra.EnableCommandSorting = false` preserves command ordering

### Package Layout
```
cmd/          - Cobra command definitions (one file per command)
pkg/
  provenance/ - Core data model (Record, schema, validation)
  config/     - .gitwhy/config.yaml management
  storage/    - Pluggable backend interface + implementations
  passthrough/ - gh CLI passthrough mechanism
  drift/      - Spec drift detection
  audit/      - Export and summary generation
```

### Passthrough Strategy
- Intercept `os.Args` in `cmd.Execute()` before Cobra processes them
- Known gitwhy commands route to Cobra; everything else shells out to `gh`
- This avoids Cobra's error-on-unknown-subcommand behavior
- `gh` must be in `$PATH` for passthrough to work

## Consequences
- Zero breakage for `gh` workflows (alias `gh=ghw` is safe)
- All ghw-specific commands add `--by`, `--intent`, `--spec`, etc. as optional flags
- Storage backends are pluggable via interface; git-notes is default
