package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

func TestExtractTicketFromBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"feature/BLUE-123-auth", "BLUE-123"},
		{"JIRA-42-fix-bug", "JIRA-42"},
		{"fix/ABC-789", "ABC-789"},
		{"main", ""},
		{"release/v2.0", ""},
		{"GH-456-something", "GH-456"},
		{"no-ticket-branch", ""},
		{"", ""},
		{"feature/TOOLONGPROJECT-99999", "TOOLONGPROJECT-99999"},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := extractTicketFromBranch(tt.branch)
			if got != tt.want {
				t.Errorf("extractTicketFromBranch(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}

func TestParseIntentFromMessage(t *testing.T) {
	tests := []struct {
		msg         string
		wantSummary string
		wantOrigin  string
	}{
		{"feat: add user login", "add user login", "spec"},
		{"fix: resolve nil pointer in auth", "resolve nil pointer in auth", "spec"},
		{"perf: optimize query", "optimize query", "spec"},
		{"chore: update dependencies", "update dependencies", "human"},
		{"docs: fix typo in readme", "fix typo in readme", "human"},
		{"refactor: extract validate function", "extract validate function", "human"},
		{"test: add unit tests", "add unit tests", "human"},
		{"style: format code", "format code", "human"},
		{"build: bump version", "bump version", "human"},
		{"ci: update workflow", "update workflow", "human"},
		{"feat(api): add rate limiting", "add rate limiting", "spec"},
		{"fix(auth)!: fix security hole", "BREAKING: fix security hole", "spec"},
		{"random message", "", ""},
		{"", "", ""},
		{"no colon here", "", ""},
		{"feat:", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			summary, origin := parseIntentFromMessage(tt.msg)
			if summary != tt.wantSummary {
				t.Errorf("parseIntentFromMessage(%q) summary = %q, want %q", tt.msg, summary, tt.wantSummary)
			}
			if origin != tt.wantOrigin {
				t.Errorf("parseIntentFromMessage(%q) origin = %q, want %q", tt.msg, origin, tt.wantOrigin)
			}
		})
	}
}

func TestParseIntentFromMessageMultiline(t *testing.T) {
	summary, origin := parseIntentFromMessage("feat: add login\n\nThis implements user login with OAuth2 support")
	if summary != "add login" {
		t.Errorf("expected summary 'add login', got %q", summary)
	}
	if origin != "spec" {
		t.Errorf("expected origin 'spec', got %q", origin)
	}
}

func TestParseIntentFromMessageCustomType(t *testing.T) {
	summary, origin := parseIntentFromMessage("bump: update to v2")
	if summary != "update to v2" {
		t.Errorf("expected summary 'update to v2', got %q", summary)
	}
	if origin != "human" {
		t.Errorf("expected origin 'human' for unknown type, got %q", origin)
	}
}

func TestCurrentBranch(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	branch := currentBranch(repo)
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

func TestCurrentBranchOutsideRepo(t *testing.T) {
	branch := currentBranch(t.TempDir())
	if branch != "" {
		t.Errorf("expected empty branch outside git repo, got %q", branch)
	}
}

func TestCurrentBranchDetachedHead(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "a.txt")
	runGitCmd(t, repo, "commit", "-m", "first")
	runGitCmd(t, repo, "checkout", "--detach", "HEAD")
	branch := currentBranch(repo)
	if branch == "" {
		t.Error("expected detached HEAD to return a ref")
	}
}

func TestParseIntentFromMessagePreservesDescription(t *testing.T) {
	summary, origin := parseIntentFromMessage("refactor(api): extract handler to separate file with retry logic")
	if summary != "extract handler to separate file with retry logic" {
		t.Errorf("expected full description, got %q", summary)
	}
	if origin != "human" {
		t.Errorf("expected origin human for refactor, got %q", origin)
	}
}

func TestCommitMessage(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	runGitCmd(t, repo, "commit", "--allow-empty", "-m", "feat: add login")
	msg := commitMessage(repo)
	if msg != "feat: add login" {
		t.Errorf("expected 'feat: add login', got %q", msg)
	}
}

func TestCommitMessageOutsideRepo(t *testing.T) {
	msg := commitMessage(t.TempDir())
	if msg != "" {
		t.Errorf("expected empty outside repo, got %q", msg)
	}
}

func TestAutoFillContextFillsGitContext(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	runGitCmd(t, repo, "checkout", "-b", "feature/BLUE-42-login")

	filePath := filepath.Join(repo, "auth.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "auth.go")
	runGitCmd(t, repo, "commit", "-m", "feat: add auth handler")

	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	autoFillContext(record, repo, false, false, nil)

	if record.Context.Ticket != "BLUE-42" {
		t.Errorf("expected ticket 'BLUE-42' from branch, got %q", record.Context.Ticket)
	}
	if record.Context.Branch != "feature/BLUE-42-login" {
		t.Errorf("expected branch 'feature/BLUE-42-login', got %q", record.Context.Branch)
	}
	if record.Intent.Summary != "add auth handler" {
		t.Errorf("expected intent 'add auth handler', got %q", record.Intent.Summary)
	}
	if record.Intent.Origin != provenance.OriginType("spec") {
		t.Errorf("expected origin 'spec' for feat, got %q", record.Intent.Origin)
	}
}

func TestAutoFillContextRespectsExplicitTicket(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	runGitCmd(t, repo, "checkout", "-b", "feature/BLUE-42-login")

	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "chore: bump")

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	record.SetContext("EXPLICIT-99", "", "")
	autoFillContext(record, repo, true, true, nil)

	if record.Context.Ticket != "EXPLICIT-99" {
		t.Errorf("expected explicit ticket to be preserved, got %q", record.Context.Ticket)
	}
}

func TestAutoFillContextPreservesExplicitIntent(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "feat: add feature")

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	record.SetIntent("explicit intent", provenance.OriginHuman, "", "")
	autoFillContext(record, repo, true, false, nil)

	if record.Intent.Summary != "explicit intent" {
		t.Errorf("expected explicit intent to be preserved, got %q", record.Intent.Summary)
	}
}

func TestAutoFillContextNonConventionalMessage(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "some random message")

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	autoFillContext(record, repo, false, false, nil)

	if record.Intent.Summary != "" {
		t.Errorf("expected no intent for non-conventional message, got %q", record.Intent.Summary)
	}
}

func TestAutoFillContextWithDetachedHead(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "feat: add x")
	runGitCmd(t, repo, "checkout", "--detach", "HEAD")

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	autoFillContext(record, repo, false, false, nil)

	if record.Context.Branch == "" {
		t.Error("expected branch to be non-empty even on detached HEAD")
	}
}

func TestGenerateSummaryFilenamesMode(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "auth.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "auth.go")
	runGitCmd(t, repo, "commit", "-m", "feat: add auth")

	cfg := &config.SummaryConfig{
		Enabled: true,
		Command: "echo",
		Mode:    config.SummaryModeFilenames,
	}
	summary := generateSummary(cfg, repo)
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if !strings.Contains(summary, "auth.go") {
		t.Errorf("expected summary to contain filenames, got %q", summary)
	}
}

func TestGenerateSummaryDisabled(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)

	summary := generateSummary(&config.SummaryConfig{Enabled: false, Command: "echo"}, repo)
	if summary != "" {
		t.Errorf("expected empty summary when disabled, got %q", summary)
	}
}

func TestGenerateSummaryNilConfig(t *testing.T) {
	summary := generateSummary(nil, "/tmp")
	if summary != "" {
		t.Errorf("expected empty summary for nil config, got %q", summary)
	}
}

func TestGenerateSummaryCommandNotFound(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "test")

	cfg := &config.SummaryConfig{
		Enabled: true,
		Command: "nonexistent-command-xyz",
		Mode:    config.SummaryModeFilenames,
	}
	summary := generateSummary(cfg, repo)
	if summary != "" {
		t.Errorf("expected empty summary for missing command, got %q", summary)
	}
}

func TestGenerateSummaryNoChangedFiles(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)

	cfg := &config.SummaryConfig{
		Enabled: true,
		Command: "echo",
		Mode:    config.SummaryModeFilenames,
	}
	summary := generateSummary(cfg, repo)
	if summary != "" {
		t.Errorf("expected empty summary with no changes, got %q", summary)
	}
}

func TestAutoFillContextFallsBackToConventionalCommitWhenLlmUnavailable(t *testing.T) {
	repo := t.TempDir()
	initTestRepo(t, repo)

	if err := os.WriteFile(filepath.Join(repo, "x.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repo, "add", "x.go")
	runGitCmd(t, repo, "commit", "-m", "feat: add x file")

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	summaryCfg := &config.SummaryConfig{
		Enabled: true,
		Command: "nonexistent-llm-xyz",
		Mode:    config.SummaryModeFilenames,
	}
	autoFillContext(record, repo, false, false, summaryCfg)

	if record.Intent.Summary != "add x file" {
		t.Errorf("expected fallback to conventional commit, got %q", record.Intent.Summary)
	}
	if record.Intent.Origin != provenance.OriginType("spec") {
		t.Errorf("expected origin 'spec' for feat, got %q", record.Intent.Origin)
	}
}

func initTestRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init", "--initial-branch=main")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test")
	runGitCmd(t, dir, "commit", "--allow-empty", "-m", "initial")
}

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
