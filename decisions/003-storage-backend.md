# ADR 003: Storage Backend Design

**Date:** 2026-06-05
**Status:** Accepted

## Context
Provenance records must be stored durably and queryably. Multiple backends are needed for different deployment scenarios (local dev, CI, enterprise audit).

## Decision
### Backend Interface
```go
type Backend interface {
    Name() string
    Store(*provenance.Record) error
    Get(ref string) (*provenance.Record, error)
    List() ([]*provenance.Record, error)
    Close() error
}
```

### Implemented Backends

#### 1. Git Notes (default)
- **Ref:** `refs/notes/gitwhy`
- **Storage:** `git notes --ref refs/notes/gitwhy add -f -m "<json>"`
- **Retrieval:** `git notes --ref refs/notes/gitwhy show <ref>`
- **Pros:** Zero infrastructure, travels with the repo, no external deps
- **Cons:** Not ideal for querying across many commits; git note operations can be slow

#### 2. File Backend
- **Storage:** `.gitwhy/records/<ref>.json`
- **Pros:** Simple, debuggable, easy to back up
- **Cons:** Only available locally, not shared without git

#### 3. Metadata Backend (future)
- For PR metadata stored in PR description body or custom properties
- Phase 1 implementation

### Factory Pattern
- `storage.NewFactory()` creates a registry
- Backends register by name
- Config selects active backend via `backend` key in `.gitwhy/config.yaml`

### Record Key Strategy
- For commits: full SHA (e.g., `4f2e1a9...`)
- For PRs: PR number as string (e.g., `"412"`)
- File backend uses `<ref>.json` as filename; git notes uses the commit SHA as the note target

## Consequences
- Default backend requires only `git` (no external services)
- Teams can switch backends with `ghw config set backend file`
- All backends implement the same interface — adding a new backend doesn't change command code
