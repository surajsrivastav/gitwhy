package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old
	return buf.String()
}

func TestIsPassthrough(t *testing.T) {
	tests := []struct {
		args     []string
		expected bool
	}{
		{[]string{}, false},
		{[]string{"commit"}, false},
		{[]string{"init"}, false},
		{[]string{"config"}, false},
		{[]string{"why"}, false},
		{[]string{"log"}, false},
		{[]string{"diff"}, false},
		{[]string{"audit"}, false},
		{[]string{"version"}, false},
		{[]string{"pr", "create"}, false},
		{[]string{"pr", "view"}, false},
		{[]string{"pr", "list"}, false},
		{[]string{"pr", "checkout"}, true},
		{[]string{"pr", "merge"}, true},
		{[]string{"issue", "list"}, true},
		{[]string{"release", "create"}, true},
		{[]string{"--version"}, false},
		{[]string{"-v"}, false},
		{[]string{"--help"}, false},
		{[]string{"-h"}, false},
		{[]string{"help"}, false},
		{[]string{"unknown"}, true},
	}

	for _, tt := range tests {
		name := strings.Join(tt.args, " ")
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			if got := isPassthrough(tt.args); got != tt.expected {
				t.Errorf("isPassthrough(%v) = %v, want %v", tt.args, got, tt.expected)
			}
		})
	}
}

func TestSetVersion(t *testing.T) {
	SetVersion("abc123", "2026-01-01")
	if commitVersion != "abc123" {
		t.Errorf("expected commitVersion 'abc123', got '%s'", commitVersion)
	}
	if buildDate != "2026-01-01" {
		t.Errorf("expected buildDate '2026-01-01', got '%s'", buildDate)
	}

	SetVersion("", "")
	if commitVersion != "dev" {
		t.Errorf("expected commitVersion 'dev' for empty input, got '%s'", commitVersion)
	}
}

func TestHasProvenanceFlags(t *testing.T) {
	commitFlag = commitFlags{}
	if hasProvenanceFlags() {
		t.Error("expected false when all flags are empty")
	}

	commitFlag = commitFlags{intent: "test"}
	if !hasProvenanceFlags() {
		t.Error("expected true when intent is set")
	}

	commitFlag = commitFlags{by: "human"}
	if !hasProvenanceFlags() {
		t.Error("expected true when by is set")
	}

	commitFlag = commitFlags{spec: "SPEC-1"}
	if !hasProvenanceFlags() {
		t.Error("expected true when spec is set")
	}

	commitFlag = commitFlags{model: "gpt-4"}
	if !hasProvenanceFlags() {
		t.Error("expected true when model is set")
	}
}

func TestHasPRProvenanceFlags(t *testing.T) {
	prCreateFlags = struct {
		intent string
		origin string
		spec   string
		agent  string
		title  string
		body   string
	}{}
	if hasPRProvenanceFlags() {
		t.Error("expected false when all PR flags are empty")
	}

	prCreateFlags.intent = "test"
	if !hasPRProvenanceFlags() {
		t.Error("expected true when intent is set")
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/org/repo/pull/412", "412"},
		{"https://github.com/org/repo/pull/1", "1"},
		{"https://github.com/org/repo/pull/abc", ""},
		{"", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := extractPRNumber(tt.url); got != tt.expected {
				t.Errorf("extractPRNumber(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestPrintWhyPanel(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")
	record.SetContext("JIRA-42", "implement exponential backoff", "claude-4")

	output := captureStdout(func() {
		printWhyPanel(record)
	})

	if !strings.Contains(output, "intent:") {
		t.Error("expected intent in output")
	}
	if !strings.Contains(output, "BLUE-317") {
		t.Error("expected spec reference in output")
	}
	if !strings.Contains(output, "agent:blue") {
		t.Error("expected attribution in output")
	}
	if !strings.Contains(output, "JIRA-42") {
		t.Error("expected ticket in output")
	}
}

func TestPrintLogAnnotation(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")

	output := captureStdout(func() {
		printLogAnnotation("abc123", record)
	})

	if !strings.Contains(output, "intent: add retry logic") {
		t.Errorf("expected intent in output, got: %s", output)
	}
	if !strings.Contains(output, "agent:blue") {
		t.Errorf("expected attribution in output, got: %s", output)
	}
	if !strings.Contains(output, "BLUE-317") {
		t.Errorf("expected spec in output, got: %s", output)
	}
}

func TestPrintFullProvenance(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")
	record.SetContext("JIRA-42", "implement exponential backoff", "claude-4")

	output := captureStdout(func() {
		printFullProvenance(record)
	})

	if !strings.Contains(output, "gitwhy provenance record") {
		t.Errorf("expected header in output, got: %s", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("expected ref in output, got: %s", output)
	}
	if !strings.Contains(output, "add retry logic") {
		t.Errorf("expected intent in output, got: %s", output)
	}
	if !strings.Contains(output, "JIRA-42") {
		t.Errorf("expected ticket in output, got: %s", output)
	}
}

func TestVersionCmd(t *testing.T) {
	SetVersion("testhash", "2026-06-01T00:00:00Z")

	output := captureStdout(func() {
		err := versionCmd.RunE(versionCmd, nil)
		if err != nil {
			t.Errorf("versionCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "ghw") {
		t.Errorf("expected version output to contain 'ghw', got: %s", output)
	}
	if !strings.Contains(output, "testhash") {
		t.Errorf("expected version output to contain commit hash, got: %s", output)
	}
}

func TestVersionCmdNoBuildDate(t *testing.T) {
	SetVersion("testhash", "")

	output := captureStdout(func() {
		err := versionCmd.RunE(versionCmd, nil)
		if err != nil {
			t.Errorf("versionCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "ghw testhash") {
		t.Errorf("expected 'ghw testhash', got: %s", output)
	}
}

func TestPrintFullProvenanceMinimal(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AttributionHuman)

	output := captureStdout(func() {
		printFullProvenance(record)
	})

	if !strings.Contains(output, "human") {
		t.Errorf("expected human attribution in output, got: %s", output)
	}
}

func TestPrintLogAnnotationMinimal(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AttributionHuman)

	output := captureStdout(func() {
		printLogAnnotation("abc123", record)
	})

	if !strings.Contains(output, "human") {
		t.Errorf("expected human attribution in output, got: %s", output)
	}
}

func TestPrintWhyPanelMinimal(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AttributionHuman)

	output := captureStdout(func() {
		printWhyPanel(record)
	})

	if !strings.Contains(output, "by:      human") {
		t.Errorf("expected human attribution, got: %s", output)
	}
}

func TestInitCmdRunE(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := initCmd.RunE(initCmd, nil)
		if err != nil {
			t.Errorf("initCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy initialized") {
		t.Errorf("expected success message, got: %s", output)
	}

	hookBytes, err := os.ReadFile(filepath.Join(tmpDir, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatal("expected post-commit hook to exist")
	}
	hook := string(hookBytes)
	if !strings.Contains(hook, "ghw commit") {
		t.Error("expected hook to contain 'ghw commit'")
	}
	if !strings.Contains(hook, "GHW_CAPTURE=1") {
		t.Error("expected hook to contain GHW_CAPTURE=1")
	}
}

func TestInitCmdNoHook(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	initCmd.Flags().Set("no-hook", "true")
	t.Cleanup(func() { initCmd.Flags().Set("no-hook", "false") })

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := initCmd.RunE(initCmd, nil)
		if err != nil {
			t.Errorf("initCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy initialized") {
		t.Errorf("expected success message, got: %s", output)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, ".git", "hooks", "post-commit")); err == nil {
		t.Error("expected no post-commit hook when --no-hook is set")
	}
}

func TestInitCmdAlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	os.MkdirAll(filepath.Join(tmpDir, ".gitwhy"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	initCmd.RunE(initCmd, nil)
}

func TestInitCmdNotInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := initCmd.RunE(initCmd, nil)
	if err == nil {
		t.Error("expected error when not in git repo")
	}
}

func TestConfigGetCmd(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := configGetCmd.RunE(configGetCmd, []string{})
		if err != nil {
			t.Errorf("configGetCmd error = %v", err)
		}
	})

	if !strings.Contains(output, "backend:") {
		t.Errorf("expected config output, got: %s", output)
	}
}

func TestConfigGetCmdWithKey(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := configGetCmd.RunE(configGetCmd, []string{"backend"})
		if err != nil {
			t.Errorf("configGetCmd error = %v", err)
		}
	})

	if !strings.Contains(output, "file") {
		t.Errorf("expected 'file', got: %s", output)
	}
}

func TestConfigGetCmdUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := configGetCmd.RunE(configGetCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestConfigSetCmd(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: git-notes\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := configSetCmd.RunE(configSetCmd, []string{"backend", "file"})
		if err != nil {
			t.Errorf("configSetCmd error = %v", err)
		}
	})

	if !strings.Contains(output, "backend = file") {
		t.Errorf("expected 'backend = file', got: %s", output)
	}
}

func TestConfigSetCmdInvalidBackend(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	os.MkdirAll(filepath.Join(tmpDir, ".gitwhy"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644)

	err := configSetCmd.RunE(configSetCmd, []string{"backend", "invalid"})
	if err == nil {
		t.Error("expected error for invalid backend")
	}
}

func TestConfigSetCmdUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := configSetCmd.RunE(configSetCmd, []string{"unknown", "value"})
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestWhyCmdNoProvenance(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)
	exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").CombinedOutput()

	out, _ := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").CombinedOutput()
	hash := strings.TrimSpace(string(out))

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := whyCmd.RunE(whyCmd, []string{hash})
	if err == nil {
		t.Error("expected error when no provenance record exists")
	}
}

func TestDiffCmdWithDrift(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("endpoint: /users\n"), 0644)

	diffFlags.drift = true
	diffFlags.spec = specPath

	output := captureStdout(func() {
		err := diffCmd.RunE(diffCmd, []string{specPath})
		if err != nil {
			t.Errorf("diffCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "drift report") {
		t.Errorf("expected drift report, got: %s", output)
	}
}

func TestDiffCmdPassthrough(t *testing.T) {
	diffFlags.drift = false
	diffFlags.spec = ""

	output := captureStdout(func() {
		diffCmd.RunE(diffCmd, nil)
	})
	_ = output
}

func TestAuditSummaryCmd(t *testing.T) {
	tmpDir := t.TempDir()

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	output := captureStdout(func() {
		_ = auditSummaryCmd.RunE(auditSummaryCmd, nil)
	})

	if !strings.Contains(output, "gitwhy audit summary") {
		t.Errorf("expected audit summary header, got: %s", output)
	}
}

func TestExecuteWithVersion(t *testing.T) {
	SetVersion("test", "2026-01-01")

	output := captureStdout(func() {
		err := versionCmd.RunE(versionCmd, nil)
		if err != nil {
			t.Errorf("versionCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "ghw test") {
		t.Errorf("expected 'ghw test', got: %s", output)
	}
}

func TestProvenanceRecordTimeAgo(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AttributionHuman)
	record.SetIntent("test", provenance.OriginHuman, "", "")

	output := captureStdout(func() {
		printFullProvenance(record)
	})

	if !strings.Contains(output, "ago") {
		t.Errorf("expected 'ago' in time output, got: %s", output)
	}
}

func TestIsPassthroughPRVariants(t *testing.T) {
	if isPassthrough([]string{"pr", "checkout"}) != true {
		t.Error("expected pr checkout to be passthrough")
	}
	if isPassthrough([]string{"pr", "merge"}) != true {
		t.Error("expected pr merge to be passthrough")
	}
	if isPassthrough([]string{"pr", "ready"}) != true {
		t.Error("expected pr ready to be passthrough")
	}
}

func TestInitCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := initCmd.RunE(initCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestExecuteWithHelp(t *testing.T) {
	output := captureStdout(func() {
		isPassthrough([]string{"--help"})
	})
	_ = output
}

func TestNewRecordHashShortening(t *testing.T) {
	shortHash := "abc1234"
	record := provenance.NewRecord(provenance.TargetCommit, shortHash)
	if record.Target.Ref != shortHash {
		t.Errorf("expected ref %q, got %q", shortHash, record.Target.Ref)
	}
}

func TestLogCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := logCmd.RunE(logCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestLogCmdWithWhy(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)
	exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").CombinedOutput()

	logFlags.why = true
	t.Cleanup(func() { logFlags.why = false })

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := logCmd.RunE(logCmd, nil)
		if err != nil {
			t.Errorf("logCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "initial") {
		t.Errorf("expected commit message in output, got: %s", output)
	}
}

func TestCommitCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := commitCmd.RunE(commitCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestAuditExportCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := auditExportCmd.RunE(auditExportCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestCommitCmdWithProvenance(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origCommitFlag := commitFlag
	commitFlag = commitFlags{
		by:     "human",
		intent: "add test file",
		spec:   "TEST-1",
	}
	t.Cleanup(func() { commitFlag = origCommitFlag })

	output := captureStdout(func() {
		err := commitCmd.RunE(commitCmd, []string{"-m", "test commit"})
		if err != nil {
			t.Errorf("commitCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy: provenance recorded") {
		t.Errorf("expected provenance recording message, got: %s", output)
	}
}

func TestCommitCmdWithoutProvenance(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origCommitFlag := commitFlag
	commitFlag = commitFlags{}
	t.Cleanup(func() { commitFlag = origCommitFlag })

	output := captureStdout(func() {
		err := commitCmd.RunE(commitCmd, []string{"-m", "test commit without provenance"})
		if err != nil {
			t.Errorf("commitCmd.RunE() error = %v", err)
		}
	})

	if strings.Contains(output, "gitwhy: provenance recorded") {
		t.Errorf("did not expect provenance recording message, got: %s", output)
	}
}

func TestCommitCmdCopilotAttribution(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origCommitFlag := commitFlag
	commitFlag = commitFlags{
		by:     "copilot",
		intent: "test copilot",
	}
	t.Cleanup(func() { commitFlag = origCommitFlag })

	output := captureStdout(func() {
		err := commitCmd.RunE(commitCmd, []string{"-m", "test copilot commit"})
		if err != nil {
			t.Errorf("commitCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy: provenance recorded") {
		t.Errorf("expected provenance recording message, got: %s", output)
	}
}

func TestCommitCmdAgentAttribution(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origCommitFlag := commitFlag
	commitFlag = commitFlags{
		by:     "agent:blue",
		intent: "test agent attribution",
	}
	t.Cleanup(func() { commitFlag = origCommitFlag })

	output := captureStdout(func() {
		err := commitCmd.RunE(commitCmd, []string{"-m", "test agent commit"})
		if err != nil {
			t.Errorf("commitCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy: provenance recorded") {
		t.Errorf("expected provenance recording message, got: %s", output)
	}
}

func TestCommitCmdWithMessageFlag(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", "test.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origCommitFlag := commitFlag
	commitFlag = commitFlags{
		by:      "human",
		intent:  "test message flag",
		message: "test message from flag",
	}
	t.Cleanup(func() { commitFlag = origCommitFlag })

	output := captureStdout(func() {
		err := commitCmd.RunE(commitCmd, nil)
		if err != nil {
			t.Errorf("commitCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "gitwhy: provenance recorded") {
		t.Errorf("expected provenance recording message, got: %s", output)
	}
}

func TestAuditExportCmdJSON(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := auditExportCmd.RunE(auditExportCmd, nil)
		if err != nil {
			t.Errorf("auditExportCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "null") {
		t.Errorf("expected empty JSON output, got: %s", output)
	}
}

func TestPrCreateCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := prCreateCmd.RunE(prCreateCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestPrViewCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := prViewCmd.RunE(prViewCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestPrListCmdNotInGit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := prListCmd.RunE(prListCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestAuditExportCmdWithDates(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	auditExportCmd.Flags().Set("from", "2026-01-01")
	auditExportCmd.Flags().Set("to", "2026-12-31")
	defer auditExportCmd.Flags().Set("from", "")
	defer auditExportCmd.Flags().Set("to", "")

	output := captureStdout(func() {
		err := auditExportCmd.RunE(auditExportCmd, nil)
		if err != nil {
			t.Errorf("auditExportCmd.RunE() with dates error = %v", err)
		}
	})

	if output == "" {
		t.Error("expected output with date filtering")
	}
}

func TestAuditExportCmdInvalidFromDate(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	auditExportCmd.Flags().Set("from", "not-a-date")
	defer auditExportCmd.Flags().Set("from", "")

	err := auditExportCmd.RunE(auditExportCmd, nil)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestAuditExportCmdInvalidToDate(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	auditExportCmd.Flags().Set("to", "not-a-date")
	defer auditExportCmd.Flags().Set("to", "")

	err := auditExportCmd.RunE(auditExportCmd, nil)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestAuditExportCmdCSVFormat(t *testing.T) {
	tmpDir := t.TempDir()
	gitCmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	configDir := filepath.Join(tmpDir, ".gitwhy")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("backend: file\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	auditExportCmd.Flags().Set("format", "csv")
	defer auditExportCmd.Flags().Set("format", "json")

	output := captureStdout(func() {
		err := auditExportCmd.RunE(auditExportCmd, nil)
		if err != nil {
			t.Errorf("auditExportCmd.RunE() CSV error = %v", err)
		}
	})

	if output == "" {
		t.Error("expected CSV output")
	}
}

func TestPrintWhyPanelWithContext(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AttributionCopilot)
	record.SetIntent("fix auth", provenance.OriginSpec, "SEC-1", "deadbeef")
	record.SetContext("JIRA-1", "prompt-here", "gpt-4")

	output := captureStdout(func() {
		fmt.Println(strings.Repeat("─", 50))
		printWhyPanel(record)
	})

	if !strings.Contains(output, "SEC-1") {
		t.Errorf("expected spec ref, got: %s", output)
	}
}
