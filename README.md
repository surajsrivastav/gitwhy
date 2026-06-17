# gitwhy (`ghw`)

Every `git commit` already knows *what* changed. gitwhy remembers *why*.

It automatically captures who (or what) made the change, which AI model was involved, the ticket from your branch name, and a one-line summary — all from a plain `git commit`. No flags, no extra commands, no thinking about it.

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
| `feature/BLUE-42-login` | Ticket `BLUE-42` — parsed from branch name |
| `feat: add login handler` | Intent `"add login handler"` — parsed from conventional commit, or summarized by your LLM CLI |
| ...with Claude open | Model `claude-sonnet-4` — auto-detected from `$CLAUDE_MODEL` |
| ...and Copilot running | Attribution `copilot` — from `--by` flag or config |

You get a record like:

```
intent:    add login handler
origin:    spec
context:
    ticket:   BLUE-42
    branch:   feature/BLUE-42-login
    model:    claude-sonnet-4
```

## Commands

| If you want to... | Run this |
|---|---|
| Set up gitwhy in a repo | `ghw init` |
| Commit with explicit flags | `ghw commit --by copilot --ticket BLUE-42` |
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
| `--ticket` | Reference: e.g. `BLUE-42` |
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

**You need:** Go 1.26.4+, Git, and the [GitHub CLI](https://cli.github.com/) (`gh`).

```bash
git clone https://github.com/surajsrivastav/gitwhy.git
cd gitwhy
make build
```

Then drop the binary somewhere in your PATH:

```bash
cp ghw /opt/homebrew/bin/   # macOS Homebrew
sudo cp ghw /usr/local/bin/ # Linux/macOS
# or: mkdir -p ~/bin && cp ghw ~/bin
```

Verify:

```bash
ghw version
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
