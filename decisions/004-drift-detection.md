# ADR 004: Drift Detection Strategy

**Date:** 2026-06-05
**Status:** Accepted

## Context
When code is generated from a spec, the implementation can drift from the spec over time. Drift detection must identify when the spec has changed since code generation.

## Decision
### Detection Modes (Phase 2 scope)
1. **Spec hash comparison** — Compare SHA-256 hash of current spec content against hash recorded at generation time
2. **Semantic diff** — AI-assisted comparison (Phase 2+, not implemented yet)
3. **Structural drift** — File structure comparison (Phase 2+, not implemented yet)

### Hash Strategy
- Full SHA-256 of spec file content, truncated to first 8 hex bytes (16 chars)
- Stored as `spec_hash` on the provenance record
- Deterministic — same content always produces same hash

### Report Structure
```go
type Report struct {
    File             string   // File being checked
    Spec             string   // Spec identifier
    GeneratedAt      string   // Timestamp of generation
    GeneratedBy      string   // Attribution
    SpecHashAtGen    string   // Hash at code generation time
    CurrentSpecHash  string   // Current hash of spec file
    DriftDetected    bool     // True if hashes differ
    Severity         Severity // none | low | medium | high
    AffectedLines    int      // Lines in spec that changed
    Recommendation   string   // Actionable recommendation
}
```

### CLI Usage
```bash
ghw diff --drift src/auth/tenant.ts --spec BLUE-204
```
This compares the current `BLUE-204` spec against the hash stored in the provenance record for that file.

## Consequences
- Hash comparison is cheap and deterministic — no LLM calls needed
- Truncated hash (16 chars) is sufficient for collision resistance at repo scale
- Drift detection requires the spec file to be accessible locally
- Phase 2 will add semantic drift using AI-assisted comparison
