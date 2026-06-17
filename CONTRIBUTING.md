# Contributing

## Setup

1. Clone the repo:
   ```sh
   git clone https://github.com/anomalyco/gitwhy.git
   cd gitwhy
   ```
2. Build:
   ```sh
   make build
   ```
3. Install (adds to `$GOPATH/bin`):
   ```sh
   go build -o ~/bin/ghw .
   # or
   make install
   ```

## Testing

```sh
make test          # full suite with -v
make test-short    # quiet run (no -v)
make vet           # go vet
make coverage      # generate coverage.html
```

### Mutation testing

```sh
go install github.com/avito-tech/go-mutesting/...@latest
./scripts/mutate.sh
```

See `PRD.md` for current mutation score targets (85%+).

## Project structure

```
cmd/           — cobra command definitions (init, commit, pr, etc.)
pkg/audit      — provenance audit and export
pkg/config     — config loading and schema
pkg/drift      — spec-drift analysis
pkg/passthrough— gh CLI passthrough and exec helpers
pkg/provenance — core schema: Record, Context, Intent
pkg/storage    — backends: git-notes, file
```

## Commit conventions

This project uses conventional commits. The post-commit hook
auto-extracts intent from commit messages:

| Type       | Origin |
|------------|--------|
| `feat`     | spec   |
| `fix`      | spec   |
| `perf`     | spec   |
| `chore`    | human  |
| `docs`     | human  |
| `refactor` | human  |
| `style`    | human  |
| `test`     | human  |
| `ci`       | human  |
| `build`    | human  |

Breaking changes use `!` before the colon: `feat!: ...`.

## Submitting a PR

1. Create a feature branch from `master`.
2. Make your changes.
3. Run `make test && make vet`.
4. Push and open a PR. Add `--intent` and `--origin` flags to
   describe your change:
   ```sh
   ghw pr create --intent "add foo bar" --origin human
   ```

## Code style

- Follow Go conventions (`gofmt`/`go vet` clean).
- No external LLM SDKs — shell out to whatever CLI the user has.
- New features must include tests.
- Keep the mutation score at 85%+

## Reporting issues

Use the GitHub issue tracker. Include:
- `ghw version` output
- Go version (`go version`)
- OS and architecture
- Steps to reproduce
