# gitwhy (`ghw`)

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Test](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/ci.yml?branch=master&style=flat-square&logo=githubactions&label=tests)](https://github.com/surajsrivastav/gitwhy/actions/workflows/ci.yml)
[![Build](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/release.yml?style=flat-square&logo=githubactions&label=build)](https://github.com/surajsrivastav/gitwhy/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/surajsrivastav/gitwhy?style=flat-square&logo=github)](https://github.com/surajsrivastav/gitwhy/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/surajsrivastav/gitwhy?style=flat-square)](https://goreportcard.com/report/github.com/surajsrivastav/gitwhy)
[![Coverage](https://codecov.io/gh/surajsrivastav/gitwhy/branch/master/graph/badge.svg)](https://codecov.io/gh/surajsrivastav/gitwhy)

Every `git commit` already knows *what* changed. gitwhy remembers *why*.

It automatically captures who (or what) made the change, which AI model was involved, the ticket from your branch name, and a one-line summary — all from a plain `git commit`. No flags, no extra commands, no thinking about it.

## Why gitwhy?

**The problem:** AI-generated code is everywhere. Copilot, Claude, Cursor — every commit is a mix of human and AI work. Months later, nobody knows which model produced what, why a change was made, or what spec it came from. Audit trails are empty. Debugging is guesswork.

**What gitwhy does:** Every `git commit` automatically captures provenance — the AI model, the ticket, the intent, the origin (human/spec/AI). Stored locally in git-notes. Zero friction. Plain `git commit` stays plain.

**Who needs this:**
- **Teams shipping AI-generated code** — prove what models produced what
- **Compliance/audit** — trace every change back to a spec or prompt
- **Debugging** — "which model wrote this?" answered in one command
- **Solo devs** — never wonder "why did I do that?" six months later

## How it works

`ghw init` installs a post-commit hook. After that, every `git commit` silently records provenance in the background. That's it.

```bash
cd your-repo
ghw init                        # one-time setup
git add . && git commit -m "feat: add login"   # business as usual
ghw why HEAD                    # see what gitwhy captured
```

## What gets captured automatically

| You write code on... | gitwhy captures... |
|---|---|
| `feature/Ticket-42-login` | Ticket `Ticket-42` — parsed from branch name |
| `feat: add login handler` | Intent `"add login handler"` — parsed from conventional commit, or summarized by your LLM CLI |
| ...with Claude open | Model `claude-sonnet-4` — auto-detected from `$CLAUDE_MODEL` |
| ...and Copilot running | Attribution `copilot` — from `--by` flag or config |

You get a record like:

```
intent:    add login handler
origin:    spec
context:
    ticket:   Ticket-42
    branch:   feature/Ticket-42-login
    model:    claude-sonnet-4
```

## Commands

| If you want to... | Run this |
|---|---|
| Set up gitwhy in a repo | `ghw init` |
| Commit with explicit flags | `ghw commit --by copilot --ticket Ticket-42` |
| See provenance for a commit | `ghw why HEAD` |
| Browse annotated history | `ghw log --why` |
| Toggle LLM summary on/off | `ghw config set summary.enabled false` |
| Change LLM command | `ghw config set summary.command claude` |
| Export all records | `ghw audit export` |

## Flags (all optional)

Pass these to `ghw commit` when you want to override auto-detection:

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

## Configuration

Tweak behavior in `.gitwhy/config.yaml`:

```yaml
backend: git-notes                # how records are stored
auto_capture:
  enabled: true
  default_by: agent:opencode      # default attribution for auto-capture
summary:
  enabled: true                   # generate intent via LLM
  command: llm                    # any CLI that takes a prompt as last arg
  mode: filenames                  # filenames | diff
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
