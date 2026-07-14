# gitwhy — Product Requirements Document

## Overview

gitwhy (`ghw`) is a CLI tool that wraps the GitHub CLI (`gh`) with a provenance and
intent layer built for the age of AI-generated code. Every commit carries structured
metadata: who (or what) authored it, why it was made, what spec or prompt drove it,
and which AI model was involved.

## Features

### 1. Provenance Recording

Every `ghw commit` stores a JSON provenance record via a configurable backend
(git-notes or file). The record captures:

- **Target**: commit hash
- **Attribution**: human, copilot, pair, or `agent:<name>`
- **Intent**: summary, spec reference, spec hash, origin type (human/spec/prompt/...)
- **Context**: ticket ID, prompt text, AI model name

### 2. Model Auto-Detection

If `--model` is not explicitly provided, `ghw commit` attempts to auto-detect the
model from the environment. **The detected model reflects what is active at commit
time**, not necessarily the model used throughout the entire coding session.

Detection sources (checked in order):

| Source | Env Variable | Example |
|--------|-------------|---------|
| Claude/Anthropic | `ANTHROPIC_MODEL` | `claude-sonnet-4-20250514` |
| OpenAI | `OPENAI_MODEL` | `gpt-4o`, `o3` |
| GitHub Copilot | `GITHUB_MODEL` | `gpt-5` |
| Claude Code | `CLAUDE_MODEL` | `claude-sonnet-4` |
| Generic AI | `AI_MODEL` | any model name |
| Interactive prompt | stdin (terminal only) | user types name |

Detection rules:
- **Single match**: auto-used silently with a confirmation message
- **Multiple matches**: candidates listed, user prompted interactively
- **No match**: user prompted interactively
- **Non-interactive** (piped stdin): prompting skipped, model left empty
- **Explicit `--model` flag**: always takes priority over detection

### 3. Commands

| Command | Description |
|---------|-------------|
| `ghw commit` | Commit with provenance flags and model auto-detection |
| `ghw why <ref>` | Show provenance record for a commit, including tamper-check status |
| `ghw init` | Initialize gitwhy in a repository |
| `ghw config` | Manage configuration |
| `ghw session start/status/set/end` | Track the model/ticket/prompt active for the current coding session |
| `ghw log` | Show commit log with intent annotations |
| `ghw pr create/view/list` | PR operations with provenance |
| `ghw diff` | Show diff with provenance context, including spec drift |
| `ghw audit export/search/summary` | Export, filter, or summarize provenance records |
| `ghw audit gaps` | List commits with no provenance record (`--fail-on-missing` for CI) |
| `ghw audit verify` | Batch-check every record's `record_hash` for tampering |

### 4. Storage Backends

- **git-notes**: Stores records as git notes, travels with the repository
- **file**: Stores records as JSON files in `.gitwhy/records/`

### 5. Drift Detection

Detects when a tracked spec has changed after code generation. `--spec` accepts
a local file path, any http(s) URL, or a GitHub issue URL (fetched via the
local `gh` CLI, no separate token needed). Produces a drift report with
severity, affected lines, and recommendations.

### 6. Session Tracking

`ghw session start --model <model> --ticket <ticket>` writes
`.gitwhy/session.yaml` (gitignored by default). The post-commit hook and
`ghw commit` read the session file before falling back to environment-variable
detection or `default_model`, so the model that actually wrote the code — not
whichever model happens to be active when `git commit` runs — gets recorded.
`ghw session status` shows the active session and its age; `ghw session set
<field> <value>` updates a single field; `ghw session end` clears it.

### 7. Coverage Gaps and Tamper Detection

`ghw audit gaps [--since <date-or-ref>] [--fail-on-missing]` diffs the commits
in range against stored provenance records and lists every commit with no
record — the question a reviewer or CI gate actually needs answered. Wiring
`--fail-on-missing` into a GitHub Actions job turns this into a merge gate.

Every record is sealed with a `record_hash` (SHA-256 of its content) at write
time. `ghw audit verify` batch-checks every record's hash and flags any that
no longer matches — evidence that the note was edited outside of gitwhy after
being recorded (e.g. a manual `git notes edit`). `ghw why` shows the
per-record verification status. This detects tampering; it does not prevent
it — anyone who can rewrite a note can also recompute a matching hash. See
the Compliance Roadmap below for what closes that gap.

### 8. Filtering and Search

`ghw audit export` and the interactive `ghw audit search` (table output)
both accept `--by`, `--model`, `--ticket`, `--origin`, and `--missing <field>`,
composable with AND logic — e.g. "every commit attributed to
`agent:claude-code` with no ticket set."

## Non-Goals

- Real-time model tracking during editing sessions (captures only commit-time
  state, unless a `ghw session` was started)
- Automatic detection of non-environment-based agent identities
- Cross-validation of model name accuracy
- Preventing tampering of stored records (see Compliance Roadmap — `record_hash`
  detects tampering, it doesn't stop a determined actor with push access)

## Future Considerations

- `--no-detect` flag to skip auto-detection entirely
- Richer agent detection via parent process or config directory inspection
- Git trailer integration for model info in commit message body
- Linear / Jira spec adapters (GitHub Issues is implemented; see Drift Detection)

## Positioning: Provenance Tracking, Not (Yet) Compliance

gitwhy is a **developer provenance tracking** tool: it answers "who/what wrote
this and why" for a single repository. It is not a compliance product today,
and documentation should not imply otherwise. Specifically, gitwhy does not
provide:

| Requirement | Status |
|---|---|
| Tamper-evident records | Partial — `record_hash` *detects* edits after the fact; it does not cryptographically prevent them (no signing) |
| Access controls on audit log | Not implemented — anyone with repo read access reads all notes |
| Retention policy enforcement | Not implemented — no TTL or deletion controls |
| Chain of custody | Not implemented — no signature or timestamp authority |
| Queryable by auditor | Implemented — `ghw audit search`/`export` with `--by`/`--model`/`--ticket`/`--origin`/`--missing` filters |
| Evidence of completeness (no gaps) | Implemented — `ghw audit gaps` |

## Compliance Roadmap (Future Tier)

A clearly-labeled "compliance" tier would need, in rough priority order:

1. **Signed records** — GPG or OIDC-based signatures over each record (reusing
   the committer's existing signing key where available), verified by
   `ghw audit verify`/`ghw why` alongside the existing `record_hash` check.
2. **Append-only storage option** — a signed commit chain (Certificate
   Transparency-style) or an external WORM store (S3 Object Lock), so a
   record can't be silently overwritten even with repo push access.
3. **Role-based access to audit records** — scoping who can read provenance
   notes independently of general repo read access.
4. **Retention policy configuration** — TTL/deletion rules for provenance
   data, configurable per-repo.
5. **Auditor export with completeness attestation** — an export format that
   states the commit range covered and asserts (via `ghw audit gaps`) that
   no commits in that range are missing records.

**Target framework:** SOC 2 Type II is the broadest-adoption framework for
developer tooling; the CC7 (system operations) and CC8 (change management)
Trust Services Criteria are the most directly relevant to provenance data and
should anchor which of the above ships first.

### Nice-to-Have: Changed Files & Diff Stats

Auto-capture the list of changed files and diff statistics (insertions/deletions)
in the provenance record. Git already stores this information in the commit
object, so the marginal value is low. However, inline capture could enable
faster queries and richer context without resolving the commit diff. Files
would be captured via `git diff-tree --root --name-only HEAD` and stats via
`git diff-tree --root --shortstat HEAD`.

### Nice-to-Have: Prompt Truncation

`ghw session start --prompt` (see Session Tracking above) already captures
the prompt into `context.prompt` for subsequent commits. Remaining
nice-to-have: prompts can be very long (thousands of tokens), so storing the
full text bloats records and slows storage/query. Auto-truncate to ~500
characters with a `...` suffix and store a SHA-256 hash of the full prompt
alongside the truncated text for integrity checks, without the full storage
cost.

### 9. LLM-Generated Summary

When `summary.enabled` is `true` (default) in `.gitwhy/config.yaml`, the
auto-capture flow passes the changed file list (or diff) through an external
LLM CLI to generate a one-line intent summary.

**Config schema:**

```yaml
summary:
  enabled: true
  command: llm              # any CLI that accepts a prompt as last arg
  mode: filenames            # filenames | diff
```

**Behavior:**

- LLM is tried first; if the command fails or returns empty, falls back to
  conventional-commit parsing (current behavior)
- `filenames` mode: prompt is `"Summarize this change in a few words: auth.go, main.go"`
- `diff` mode: prompt includes the actual diff output for each changed file
- Origin is set to `ai` when LLM-generated, preserving the audit trail
- Configurable per-repo via `ghw config set summary.command "claude"`
