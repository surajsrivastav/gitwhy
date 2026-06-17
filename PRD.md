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
| `ghw why <ref>` | Show provenance record for a commit |
| `ghw init` | Initialize gitwhy in a repository |
| `ghw config` | Manage configuration |
| `ghw log` | Show commit log with intent annotations |
| `ghw pr create/view/list` | PR operations with provenance |
| `ghw diff` | Show diff with provenance context |
| `ghw audit export/summary` | Export or summarize provenance records |

### 4. Storage Backends

- **git-notes**: Stores records as git notes, travels with the repository
- **file**: Stores records as JSON files in `.gitwhy/records/`

### 5. Drift Detection

Detects when a tracked spec file has changed after code generation.
Produces a drift report with severity, affected lines, and recommendations.

## Non-Goals

- Real-time model tracking during editing sessions (captures only commit-time state)
- Automatic detection of non-environment-based agent identities
- Cross-validation of model name accuracy

## Future Considerations

- `--no-detect` flag to skip auto-detection entirely
- Session tracking (`ghw start`) to capture model at coding time
- Richer agent detection via parent process or config directory inspection
- Git trailer integration for model info in commit message body

### Nice-to-Have: Changed Files & Diff Stats

Auto-capture the list of changed files and diff statistics (insertions/deletions)
in the provenance record. Git already stores this information in the commit
object, so the marginal value is low. However, inline capture could enable
faster queries and richer context without resolving the commit diff. Files
would be captured via `git diff-tree --root --name-only HEAD` and stats via
`git diff-tree --root --shortstat HEAD`.
