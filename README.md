> **Beta** — gitwhy is under active development. Feedback welcome via [GitHub Issues](https://github.com/surajsrivastav/gitwhy/issues).

# gitwhy (`ghw`)

[![Go Version](https://img.shields.io/github/go-mod/go-version/surajsrivastav/gitwhy?style=flat-square&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Test](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/ci.yml?branch=master&style=flat-square&logo=githubactions&label=tests)](https://github.com/surajsrivastav/gitwhy/actions/workflows/ci.yml)
[![Build](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/release.yml?style=flat-square&logo=githubactions&label=build)](https://github.com/surajsrivastav/gitwhy/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/surajsrivastav/gitwhy?style=flat-square&logo=github)](https://github.com/surajsrivastav/gitwhy/releases)
[![Stars](https://img.shields.io/github/stars/surajsrivastav/gitwhy?style=flat-square)](https://github.com/surajsrivastav/gitwhy/stargazers)

Every `git commit` already knows *what* changed. gitwhy remembers *why*.

It drops a single post-commit hook into your repo. That hook is a passive listener — it catches the environment variables AI tools leave behind (`$CLAUDE_CODE_MODEL`, `$COPILOT_MODEL`, `$AI_AGENT`), parses your branch name for a ticket, reads your commit message for intent, and writes a provenance record to git-notes. All without you typing a single flag.

**Two steps. Zero flags. Plain `git commit`.**

## Quick start

```bash
ghw init
git commit -m "feat: add login handler"   # standard git, nothing special

# Later — what actually happened here?
ghw why HEAD
```

```
  gitwhy provenance record
  ─────────────────────────
  schema:    gitwhy/v1
  target:    commit a3f1d8c
  by:        agent:claude-code
  when:      2026-06-25T10:00:00Z

  intent:    add login handler
  origin:    spec

  context:
    ticket:   PROJ-42
    prompt:   unknown
    model:    claude-sonnet-4-6
    branch:   feature/PROJ-42-login
```

That's the entire default workflow. `ghw init` once, then never think about it again.

## How the hook works (the Agent Interceptor)

`ghw init` installs a single script into `.git/hooks/post-commit`. Every time you (or an AI agent) run `git commit`, that hook fires automatically and silently captures:

| What it sniffs | How |
|---|---|
| **Who / what agent** | Sniffs `$AI_AGENT`, `$COPILOT_AGENT_MODEL` — detects Claude Code, Copilot, Cursor, etc. |
| **Which model** | Reads `$CLAUDE_CODE_MODEL`, `$COPILOT_MODEL`, `$ANTHROPIC_MODEL`, `$OPENAI_MODEL` |
| **Why (intent)** | Pulls the subject line from your commit message |
| **Ticket number** | Scans `git branch` for `PROJ-123` patterns |
| **Origin (human vs spec)** | Infers from conventional commit type (`feat` → `spec`, `chore` → `human`) |
| **Prompt (if AI)** | Captures `$COPILOT_AGENT_PROMPT` or `$CLAUDE_CODE_PROMPT` |

All of these environment variables are ephemeral — they exist only while the AI tool is running and vanish the moment the process exits. The hook intercepts them before they disappear.

## What gets captured automatically

Just write a normal commit message. gitwhy parses it without any flags.

```bash
git commit -m "feat: add login handler"
```

```
intent:    add login handler      ← from commit message description
origin:    spec                   ← inferred from "feat" type
ticket:    PROJ-42                ← parsed from branch feature/PROJ-42-login
model:     claude-sonnet-4-6      ← detected from $COPILOT_MODEL / $CLAUDE_CODE_MODEL
by:        agent:claude-code      ← detected from $AI_AGENT env var
branch:    feature/PROJ-42-login  ← from git
```

### Commit message → intent

gitwhy reads [conventional commit](https://www.conventionalcommits.org/) format and maps it to provenance fields automatically:

| Commit message | intent | origin |
|---|---|---|
| `feat: add login handler` | `add login handler` | `spec` |
| `fix: null pointer in auth` | `null pointer in auth` | `spec` |
| `perf: cache token lookup` | `cache token lookup` | `spec` |
| `chore: update deps` | `update deps` | `human` |
| `docs: add API examples` | `add API examples` | `human` |
| `test: cover edge cases` | `cover edge cases` | `human` |
| `feat!: breaking auth change` | `BREAKING: breaking auth change` | `spec` |

Non-conventional messages fall back to LLM summarization (if configured) then `"unknown"`.

### Branch name → ticket

gitwhy scans the branch name for a `PROJECT-123` pattern and sets it as the ticket automatically. No flags needed.

```
feature/PROJ-42-login   →  ticket: PROJ-42
fix/AUTH-7-token-null   →  ticket: AUTH-7
main                    →  ticket: unknown
```

### Agent and model → attribution

gitwhy reads environment variables set by AI tools at commit time:

| Tool | Env var read | Captured as |
|---|---|---|
| Claude Code | `AI_AGENT=claude-code/...` | `by: agent:claude-code` |
| Claude Code | `CLAUDE_CODE_MODEL` | `model: claude-sonnet-4-6` |
| GitHub Copilot CLI | `COPILOT_AGENT_MODEL` or `COPILOT_MODEL` | `by: copilot`, `model: gpt-4o` |
| GitHub Copilot CLI | `COPILOT_AGENT_PROMPT` | `prompt: ...` |
| Any tool | `ANTHROPIC_MODEL`, `OPENAI_MODEL`, `GITHUB_MODEL`, `AI_MODEL` | `model: ...` |

If no env var is found, `model` falls back to `default_model` in `.gitwhy/config.yaml`, then `"unknown"`.

Set a default model once, and every commit picks it up:

```bash
ghw config set default_model claude-sonnet-4-6
```

## Why not just use `git log` or `git blame`?

| | `git log` | `git blame` | **gitwhy** |
|---|---|---|---|
| Shows what changed | ✅ | ✅ | ✅ |
| Shows who changed it | ✅ | ✅ | ✅ |
| Shows **why** it changed | ❌ | ❌ | ✅ |
| Captures AI model used | ❌ | ❌ | ✅ |
| Links ticket/spec | ❌ | ❌ | ✅ |
| Distinguishes human vs AI | ❌ | ❌ | ✅ |
| Zero-friction (plain `git commit`) | ✅ | ✅ | ✅ |

`git log` and `git blame` tell you the **what** and **who**. gitwhy adds the **why**, **what model**, and **what spec** — the context that matters six months later when you're debugging AI-generated code.

## Commands

| If you want to... | Run this |
|---|---|
| Set up the hook in a repo | `ghw init` |
| Check hook health and last capture | `ghw status` |
| See provenance for a commit | `ghw why HEAD` |
| Browse annotated history | `ghw log --why` |
| Track the model/ticket for this coding session | `ghw session start --model claude-sonnet-4-6 --ticket PROJ-42` |
| Export all records | `ghw audit export` |
| Filter/search records | `ghw audit search --model claude-sonnet-4-6` |
| Find commits with no provenance record | `ghw audit gaps --fail-on-missing` |
| Check records for tampering | `ghw audit verify` |
| Set default model | `ghw config set default_model claude-sonnet-4-6` |
| Toggle LLM summary | `ghw config set summary.enabled false` |

> gitwhy tracks **developer provenance**, not certified compliance. See
> [PRD.md](PRD.md#positioning-provenance-tracking-not-yet-compliance) for what
> a compliance-ready tier would still need.

## Configuration

Tweak behavior in `.gitwhy/config.yaml`:

```yaml
backend: git-notes
auto_capture:
  enabled: true
  default_by: agent:opencode
summary:
  enabled: true
  command: llm
  mode: filenames
```

## Advanced: Manual flags / overrides

Everything above happens automatically. But if you ever need to override what the hook captured — for an edge case, a CI commit, or a manual annotation — pass flags to `ghw commit`:

| Flag | What it does |
|---|---|
| `--by` | Who: `human`, `copilot`, `agent:<name>` |
| `--intent` | Why: one-line description |
| `--origin` | Source: `human`, `spec`, `prompt`, `template`, `upstream` |
| `--ticket` | Reference: e.g. `Ticket-42` |
| `--spec` | Spec driving the change |
| `--spec-hash` | Spec content hash |
| `--prompt` | Prompt text (if AI-generated) |
| `--model` | Model name (overrides env detection) |
| `-m / --message` | Commit message |

```bash
# Override auto-detected values for a specific commit
ghw commit --by human --intent "harden auth middleware" --ticket SEC-99
```

## Install

**Prerequisites:** [Git](https://git-scm.com/), the [GitHub CLI](https://cli.github.com/) (`gh`), and optionally [Go](https://go.dev/dl/) 1.21+ for building from source.

### macOS (Homebrew)

```bash
brew install surajsrivastav/tap/ghw
```

### Pre-built binary (any OS)

Download the latest release for your platform from the [releases page](https://github.com/surajsrivastav/gitwhy/releases):

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/surajsrivastav/gitwhy/releases/latest/download/gitwhy_darwin_arm64.tar.gz | tar xz
sudo mv ghw /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/surajsrivastav/gitwhy/releases/latest/download/gitwhy_darwin_amd64.tar.gz | tar xz
sudo mv ghw /usr/local/bin/

# Linux (x86_64)
curl -sL https://github.com/surajsrivastav/gitwhy/releases/latest/download/gitwhy_linux_amd64.tar.gz | tar xz
sudo mv ghw /usr/local/bin/

# Linux (ARM64)
curl -sL https://github.com/surajsrivastav/gitwhy/releases/latest/download/gitwhy_linux_arm64.tar.gz | tar xz
sudo mv ghw /usr/local/bin/

# Windows (PowerShell)
curl -sLO https://github.com/surajsrivastav/gitwhy/releases/latest/download/gitwhy_windows_amd64.zip
Expand-Archive gitwhy_windows_amd64.zip -DestinationPath ~\bin
```

### Via Go (if you have Go installed)

```bash
go install github.com/surajsrivastav/gitwhy@latest
```

### Or build from source

```bash
git clone https://github.com/surajsrivastav/gitwhy.git
cd gitwhy
make build
sudo mv ghw /usr/local/bin/
```

### Install script

```bash
curl -sSfL https://raw.githubusercontent.com/surajsrivastav/gitwhy/master/install.sh | sh
```

## Project structure

```
cmd/          - CLI commands
pkg/
  provenance/ - What a record looks like
  config/     - Reading/writing .gitwhy/config.yaml
  storage/    - Where records live (git-notes or files)
  drift/      - Tracking spec changes over time
  audit/      - Reports and exports
  passthrough/ - Handing unknown commands to `gh`
```

## Testing

```bash
make test       # run all tests
make coverage   # coverage report
make vet        # check for issues
```

## License

MIT — see [LICENSE](LICENSE).

## Questions?

Check the [PRD](PRD.md) for detailed specs, or open an issue on GitHub.
