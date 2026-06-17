# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- GitHub Actions CI workflow (push/PR on Linux + macOS)
- CONTRIBUTING.md with setup, test, and submission guide
- Issue templates (bug report, feature request)
- Pull request template
- Release targets in Makefile (`release`, `snapshot`)
- `gh` CLI availability check in PR commands
- Post-commit hook falls back to PATH lookup before stored binary path

### Changed
- Hook script tries `ghw` from PATH first, then stored absolute path

## [0.1.0] - 2026-06-16

### Added
- Initial release.
- `ghw init` — scaffold config and install auto-capture post-commit hook.
- `ghw commit` — capture provenance records with model auto-detection
  and auto-context extraction (branch, ticket, intent).
- `ghw pr create/view/list` — PR management with intent capture.
- `ghw why` — display provenance for any commit or PR.
- `ghw log` — list commits with optional provenance panel.
- `ghw diff` — compare spec and implementation (drift analysis).
- `ghw audit summary/export` — provenance audit and JSON export.
- Storage backends: git-notes, file.
- Model auto-detection (env vars: `ANTHROPIC_MODEL`, `OPENAI_MODEL`,
  `CLAUDE_MODEL`, `GITHUB_MODEL`, `AI_MODEL`).
- LLM-generated intent summaries via external CLI (`llm` default).
- Conventional commit parsing for auto-origin detection.
