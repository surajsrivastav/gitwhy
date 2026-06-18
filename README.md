# gitwhy (`ghw`)

[![Go Version](https://img.shields.io/github/go-mod/go-version/surajsrivastav/gitwhy?style=flat-square&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Test](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/ci.yml?branch=master&style=flat-square&logo=githubactions&label=tests)](https://github.com/surajsrivastav/gitwhy/actions/workflows/ci.yml)
[![Build](https://img.shields.io/github/actions/workflow/status/surajsrivastav/gitwhy/release.yml?style=flat-square&logo=githubactions&label=build)](https://github.com/surajsrivastav/gitwhy/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/surajsrivastav/gitwhy?style=flat-square&logo=github)](https://github.com/surajsrivastav/gitwhy/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/surajsrivastav/gitwhy?style=flat-square)](https://goreportcard.com/report/github.com/surajsrivastav/gitwhy)
[![Coverage](https://codecov.io/gh/surajsrivastav/gitwhy/branch/master/graph/badge.svg)](https://codecov.io/gh/surajsrivastav/gitwhy)

Your `git log` shows *what* changed. gitwhy captures *why*.

AI tools write a lot of your code now. Copilot, Claude, Cursor — six months later, nobody can tell which model wrote what, what ticket drove the change, or whether a human even looked at it. `git blame` just points at a name and a date.

gitwhy fixes that. After a one-time `ghw init`, every `git commit` silently records the full picture in the background: which AI model was involved, which ticket it was for, what the intent was, and whether a human or a spec drove it. No new commands. No new habits. Plain `git commit` stays plain.

## Quickstart

```bash
brew install surajsrivastav/tap/ghw   # or see other install options below

cd your-repo
ghw init                              # one-time setup, installs a post-commit hook

git add . && git commit -m "feat: add login"   # nothing changes here
ghw why HEAD                          # see what gitwhy just captured
```

That's it. From here on every commit carries its own provenance record.

## What gets recorded

| When you commit from... | gitwhy captures... |
|---|---|
| `feature/PROJ-42-login` | Ticket `PROJ-42` — parsed from branch name |
| `feat: add login handler` | Intent `"add login handler"` — from the commit message, or summarised by your LLM CLI |
| ...with Claude open | Model `claude-sonnet-4` — auto-detected from `$CLAUDE_MODEL` |
| ...and Copilot running | Attribution `copilot` — from `--by` or config |

The result looks like this:

```
intent:    add login handler
origin:    spec
context:
    ticket:   PROJ-42
    branch:   feature/PROJ-42-login
    model:    claude-sonnet-4
```

Stored in git-notes alongside the commit. No extra files, no external service, no database.

## Commands

| What you want | Command |
|---|---|
| Set up gitwhy in a repo | `ghw init` |
| Commit with explicit attribution | `ghw commit --by copilot --ticket PROJ-42` |
| See why a commit was made | `ghw why HEAD` |
| Browse history with provenance | `ghw log --why` |
| Turn off LLM summaries | `ghw config set summary.enabled false` |
| Switch LLM command | `ghw config set summary.command claude` |
| Export all records to JSON | `ghw audit export` |

## Optional flags

Everything is auto-detected, but you can override any field when committing:

| Flag | What it sets |
|---|---|
| `--by` | Who: `human`, `copilot`, `agent:<name>` |
| `--intent` | Why: one-line description |
| `--origin` | Source: `human`, `spec`, `prompt`, `template`, `upstream` |
| `--ticket` | Ticket reference, e.g. `PROJ-42` |
| `--spec` | Spec driving the change |
| `--prompt` | Prompt text if AI-generated |
| `--model` | Model name (overrides env detection) |
| `-m / --message` | Commit message |

## Configuration

`.gitwhy/config.yaml` in your repo:

```yaml
backend: git-notes          # git-notes (default) or file
auto_capture:
  enabled: true
  default_by: agent:claude  # default attribution for auto-captured commits
summary:
  enabled: true             # generate intent summary via LLM
  command: llm              # any CLI that accepts a prompt as its last argument
  mode: filenames           # filenames | diff
```

## Install

**Prerequisites:** [Git](https://git-scm.com/) and the [GitHub CLI](https://cli.github.com/) (`gh`).

### macOS (Homebrew) — recommended

```bash
brew install surajsrivastav/tap/ghw
```

### Go install

```bash
go install github.com/surajsrivastav/gitwhy@latest
```

### Install script

```bash
curl -sSfL https://raw.githubusercontent.com/surajsrivastav/gitwhy/master/install.sh | sh
```

### Pre-built binaries

Download for your platform from the [releases page](https://github.com/surajsrivastav/gitwhy/releases):

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

### Build from source

```bash
git clone https://github.com/surajsrivastav/gitwhy.git
cd gitwhy
make build
sudo mv ghw /usr/local/bin/
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Run `make test` before opening a PR.

## License

MIT — see [LICENSE](LICENSE).
