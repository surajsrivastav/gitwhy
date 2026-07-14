package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

// initGitwhyConfigOnly writes .gitwhy/config.yaml directly, without going
// through initCmd (which installs a real post-commit hook pointing at the
// test binary — triggering a real `git commit` in-process would then
// re-invoke the test binary recursively).
func initGitwhyConfigOnly(t *testing.T, repoPath string) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.RepoPath = repoPath
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)
	return tmpDir
}

func commitFile(t *testing.T, tmpDir, name, content, message string) string {
	t.Helper()
	filePath := filepath.Join(tmpDir, name)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "add", name).CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", tmpDir, "commit", "-m", message).CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, err := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestSessionStartStatusSetEnd(t *testing.T) {
	tmpDir := newGitRepo(t)
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origFlags := sessionStartFlags
	sessionStartFlags = struct {
		model  string
		ticket string
		prompt string
		origin string
	}{model: "claude-sonnet-4-6", ticket: "JIRA-1"}
	t.Cleanup(func() { sessionStartFlags = origFlags })

	if err := sessionStartCmd.RunE(sessionStartCmd, nil); err != nil {
		t.Fatalf("session start: %v", err)
	}

	output := captureStdout(func() {
		if err := sessionStatusCmd.RunE(sessionStatusCmd, nil); err != nil {
			t.Fatalf("session status: %v", err)
		}
	})
	if !strings.Contains(output, "claude-sonnet-4-6") || !strings.Contains(output, "JIRA-1") {
		t.Errorf("expected session status to show model/ticket, got: %s", output)
	}

	if err := sessionSetCmd.RunE(sessionSetCmd, []string{"model", "claude-opus-4-7"}); err != nil {
		t.Fatalf("session set: %v", err)
	}
	session, err := config.LoadSession(tmpDir)
	if err != nil || session == nil || session.Model != "claude-opus-4-7" {
		t.Fatalf("expected updated model, got session=%+v err=%v", session, err)
	}

	if err := sessionEndCmd.RunE(sessionEndCmd, nil); err != nil {
		t.Fatalf("session end: %v", err)
	}
	session, err = config.LoadSession(tmpDir)
	if err != nil {
		t.Fatalf("load session after end: %v", err)
	}
	if session != nil {
		t.Errorf("expected session to be cleared, got %+v", session)
	}
}

func TestSessionStatusNoSession(t *testing.T) {
	tmpDir := newGitRepo(t)
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		if err := sessionStatusCmd.RunE(sessionStatusCmd, nil); err != nil {
			t.Fatalf("session status: %v", err)
		}
	})
	if !strings.Contains(output, "no active session") {
		t.Errorf("expected no active session message, got: %s", output)
	}
}

func TestSessionSetUnknownField(t *testing.T) {
	tmpDir := newGitRepo(t)
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	if err := sessionSetCmd.RunE(sessionSetCmd, []string{"bogus", "value"}); err == nil {
		t.Error("expected error for unknown session field")
	}
}

func TestAuditGapsFindsMissingCommits(t *testing.T) {
	tmpDir := newGitRepo(t)
	if err := os.Mkdir(filepath.Join(tmpDir, ".gitwhy"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644); err != nil {
		t.Fatal(err)
	}

	commitFile(t, tmpDir, "a.txt", "a", "first commit")
	commitFile(t, tmpDir, "b.txt", "b", "second commit")

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	origFlags := commitFlag
	t.Cleanup(func() { commitFlag = origFlags })

	output := captureStdout(func() {
		if err := auditGapsCmd.RunE(auditGapsCmd, nil); err != nil {
			t.Fatalf("audit gaps: %v", err)
		}
	})
	if !strings.Contains(output, "2 commits missing provenance records (out of 2 total)") {
		t.Errorf("expected 2 gaps reported, got: %s", output)
	}
}

func TestAuditGapsFailOnMissing(t *testing.T) {
	tmpDir := newGitRepo(t)
	if err := os.Mkdir(filepath.Join(tmpDir, ".gitwhy"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commitFile(t, tmpDir, "a.txt", "a", "first commit")

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	if err := auditGapsCmd.Flags().Set("fail-on-missing", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auditGapsCmd.Flags().Set("fail-on-missing", "false") })

	captureStdout(func() {
		err := auditGapsCmd.RunE(auditGapsCmd, nil)
		if err == nil {
			t.Error("expected error with --fail-on-missing when gaps exist")
		}
		if _, ok := err.(*exitCodeError); !ok {
			t.Errorf("expected *exitCodeError, got %T: %v", err, err)
		}
	})
}

func TestAuditGapsNoGapsWhenAllRecorded(t *testing.T) {
	tmpDir := newGitRepo(t)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	initGitwhyConfigOnly(t, tmpDir)

	commitFile(t, tmpDir, "a.txt", "a", "first commit")

	origFlags := commitFlag
	commitFlag = commitFlags{by: "human", intent: "test"}
	t.Cleanup(func() { commitFlag = origFlags })

	os.Setenv("GHW_CAPTURE", "1")
	defer os.Unsetenv("GHW_CAPTURE")
	if err := commitCmd.RunE(commitCmd, nil); err != nil {
		t.Fatalf("commit capture: %v", err)
	}

	output := captureStdout(func() {
		if err := auditGapsCmd.RunE(auditGapsCmd, nil); err != nil {
			t.Fatalf("audit gaps: %v", err)
		}
	})
	if !strings.Contains(output, "no gaps") {
		t.Errorf("expected no gaps, got: %s", output)
	}
}

func TestAuditSearchAndVerify(t *testing.T) {
	tmpDir := newGitRepo(t)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	initGitwhyConfigOnly(t, tmpDir)

	commitFile(t, tmpDir, "a.txt", "a", "first commit")

	origFlags := commitFlag
	commitFlag = commitFlags{by: "human", intent: "test", model: "claude-sonnet-4-6", ticket: "JIRA-9"}
	t.Cleanup(func() { commitFlag = origFlags })

	os.Setenv("GHW_CAPTURE", "1")
	defer os.Unsetenv("GHW_CAPTURE")
	if err := commitCmd.RunE(commitCmd, nil); err != nil {
		t.Fatalf("commit capture: %v", err)
	}

	searchOut := captureStdout(func() {
		if err := auditSearchCmd.RunE(auditSearchCmd, nil); err != nil {
			t.Fatalf("audit search: %v", err)
		}
	})
	if !strings.Contains(searchOut, "JIRA-9") {
		t.Errorf("expected ticket in search output, got: %s", searchOut)
	}

	verifyOut := captureStdout(func() {
		if err := auditVerifyCmd.RunE(auditVerifyCmd, nil); err != nil {
			t.Fatalf("audit verify: %v", err)
		}
	})
	if !strings.Contains(verifyOut, "1 records checked, 0 tampered") {
		t.Errorf("expected verify summary, got: %s", verifyOut)
	}
}

func TestAuditSearchWithModelFilter(t *testing.T) {
	tmpDir := newGitRepo(t)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	initGitwhyConfigOnly(t, tmpDir)

	commitFile(t, tmpDir, "a.txt", "a", "first commit")

	origFlags := commitFlag
	commitFlag = commitFlags{by: "human", intent: "test", model: "claude-sonnet-4-6"}
	t.Cleanup(func() { commitFlag = origFlags })

	os.Setenv("GHW_CAPTURE", "1")
	defer os.Unsetenv("GHW_CAPTURE")
	if err := commitCmd.RunE(commitCmd, nil); err != nil {
		t.Fatalf("commit capture: %v", err)
	}

	if err := auditSearchCmd.Flags().Set("model", "claude-opus-4-7"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auditSearchCmd.Flags().Set("model", "") })

	output := captureStdout(func() {
		if err := auditSearchCmd.RunE(auditSearchCmd, nil); err != nil {
			t.Fatalf("audit search: %v", err)
		}
	})
	if !strings.Contains(output, "no matching records") {
		t.Errorf("expected no matches for wrong model filter, got: %s", output)
	}
}
