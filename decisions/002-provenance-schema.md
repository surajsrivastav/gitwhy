# ADR 002: Provenance Data Model (Schema v1)

**Date:** 2026-06-05
**Status:** Accepted

## Context
Provenance records must store structured metadata about who/what authored a change, why it was made, and what spec drove it. The schema must be versioned for forward compatibility.

## Decision
### Schema Structure
```json
{
  "schema": "gitwhy/v1",
  "target": { "type": "commit|pr", "ref": "<sha or pr number>" },
  "attribution": { "by": "human|copilot|agent:<name>|pair", "timestamp": "<ISO 8601>" },
  "intent": { "summary": "...", "spec": "...", "spec_hash": "...", "origin": "..." },
  "context": { "ticket": "...", "prompt": "...", "model": "..." }
}
```

### Key Decisions
- **`attribution.by` uses `agent:<name>` format** — extensible for any AI agent without schema changes
- **`spec_hash` is SHA-256 truncated to 8 hex chars** — enough for drift detection, short enough for CLI display
- **Timestamp is RFC 3339** — standard, sortable, timezone-aware
- **Context fields are all optional** — provenance is additive, never blocking
- **Validation is strict** — schema version, target type, attribution, and ref are required

### `IsAgentAttribution()` helper
- Checks prefix `agent:` — this is the canonical way to detect AI authorship
- `copilot` and `human` are non-agent attributions
- `pair` is reserved for human-pair programming

## Consequences
- Schema v1 is fixed for Phase 0/1; `schema` field allows migration to v2+
- Validation prevents corrupt records from being stored
- JSON serialization is the interchange format across all storage backends
