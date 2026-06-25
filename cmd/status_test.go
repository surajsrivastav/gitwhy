package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

// setupGitRepo initialises a minimal git repo at tmpDir.
func setupGitRepo(t *testing.T, tmpDir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// TestStatusCmdNoInit — running ghw status outside a git repo returns an error.
func TestStatusCmdNoInit(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err := statusCmd.RunE(statusCmd, nil)
	if err == nil {
		t.Error("expected error when not in a git repository")
	}
}

// TestStatusCmdNoCapture — initialized repo with hook but no last-capture shows "never".
func TestStatusCmdNoCapture(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Install hook and write config so it looks initialized.
	if err := installHook(tmpDir); err != nil {
		t.Fatalf("installHook() error = %v", err)
	}
	os.MkdirAll(filepath.Join(tmpDir, ".gitwhy"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := statusCmd.RunE(statusCmd, nil)
		if err != nil {
			t.Errorf("statusCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "never") {
		t.Errorf("expected 'never' in output for no last-capture, got: %s", output)
	}
	if !strings.Contains(output, "installed") {
		t.Errorf("expected hook 'installed' in output, got: %s", output)
	}
}

// TestStatusCmdWithCapture — after writing a receipt, shows "X minutes ago (hash)".
func TestStatusCmdWithCapture(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".gitwhy"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644)

	if err := config.WriteLastCapture(tmpDir, "9f54657abcdef", "git-notes"); err != nil {
		t.Fatalf("WriteLastCapture() error = %v", err)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := statusCmd.RunE(statusCmd, nil)
		if err != nil {
			t.Errorf("statusCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "ago") {
		t.Errorf("expected 'ago' in output for recent capture, got: %s", output)
	}
	// Short hash (first 7 chars).
	if !strings.Contains(output, "9f54657") {
		t.Errorf("expected short hash '9f54657' in output, got: %s", output)
	}
}

// TestStatusCmdWithErrors — after writing errors, shows error count.
func TestStatusCmdWithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".gitwhy"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".gitwhy", "config.yaml"), []byte("backend: git-notes\n"), 0644)

	if err := config.AppendCaptureError(tmpDir, "abc1234", "backend unavailable"); err != nil {
		t.Fatalf("AppendCaptureError() error = %v", err)
	}
	if err := config.AppendCaptureError(tmpDir, "def5678", "timeout"); err != nil {
		t.Fatalf("AppendCaptureError() error = %v", err)
	}
	if err := config.AppendCaptureError(tmpDir, "ghi9012", "network error"); err != nil {
		t.Fatalf("AppendCaptureError() error = %v", err)
	}

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	output := captureStdout(func() {
		err := statusCmd.RunE(statusCmd, nil)
		if err != nil {
			t.Errorf("statusCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "3") {
		t.Errorf("expected error count '3' in output, got: %s", output)
	}
	if !strings.Contains(output, config.CaptureErrorsFile) {
		t.Errorf("expected capture-errors filename in output, got: %s", output)
	}
}
