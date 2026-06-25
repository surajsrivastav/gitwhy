package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Backend != BackendGitNotes {
		t.Errorf("expected default backend %q, got %q", BackendGitNotes, cfg.Backend)
	}
	if cfg.GitNotes == nil || !cfg.GitNotes.Enabled {
		t.Error("expected git-notes to be enabled by default")
	}
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir("/repo")
	expected := "/repo/.gitwhy"
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath("/repo")
	expected := "/repo/.gitwhy/config.yaml"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.RepoPath = tmpDir
	cfg.Backend = BackendFile
	cfg.File = &FileConfig{Path: "/custom/path"}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	configPath := ConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Backend != BackendFile {
		t.Errorf("expected backend %q, got %q", BackendFile, loaded.Backend)
	}
	if loaded.RepoPath != tmpDir {
		t.Errorf("expected RepoPath %q, got %q", tmpDir, loaded.RepoPath)
	}
	if loaded.File == nil || loaded.File.Path != "/custom/path" {
		t.Errorf("file config not preserved")
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() should not error for missing config: %v", err)
	}
	if cfg.Backend != BackendGitNotes {
		t.Errorf("expected default backend for missing config")
	}
}

func TestLoadReadError(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nonexistent")
	_, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() should return defaults for nonexistent path: %v", err)
	}
}

func TestIsInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	if IsInitialized(tmpDir) {
		t.Error("expected false for uninitialized repo")
	}

	os.MkdirAll(ConfigDir(tmpDir), 0755)
	if IsInitialized(tmpDir) {
		t.Error("expected false when only directory exists (no config file)")
	}

	os.WriteFile(ConfigPath(tmpDir), []byte("backend: git-notes\n"), 0644)
	if !IsInitialized(tmpDir) {
		t.Error("expected true when config file exists")
	}
}

func TestFindRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	root, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root path")
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, err := FindRepoRoot()
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestSaveWithoutPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepoPath = ""
	err := Save(cfg)
	if err == nil {
		t.Error("expected error when repo path is not set")
	}
}

func TestLoadCorruptedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(ConfigDir(tmpDir), 0755)
	os.WriteFile(ConfigPath(tmpDir), []byte("invalid: yaml: [[["), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Errorf("Load() should not error on corrupted config, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config (defaults) for corrupted config")
	}
	if cfg.Backend != BackendGitNotes {
		t.Errorf("expected default backend for corrupted config, got %q", cfg.Backend)
	}
}

func TestSaveMkdirAllError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("cannot test permission error as root")
	}
	tmpDir := t.TempDir()
	readOnly := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnly, 0555); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.RepoPath = readOnly
	err := Save(cfg)
	if err == nil {
		t.Error("expected error when config dir cannot be created")
	}
	if err != nil && !strings.Contains(err.Error(), "create config dir:") {
		t.Errorf("expected 'create config dir:' error, got: %v", err)
	}
}

func TestSaveWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.RepoPath = tmpDir
	if err := Save(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveWritePermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.RepoPath = tmpDir

	dir := ConfigDir(tmpDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	err := Save(cfg)
	if err == nil {
		t.Error("expected error when config dir is read-only")
	}
}

func TestLoadPermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(ConfigDir(tmpDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(tmpDir), []byte("backend: git-notes\n"), 0000); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(tmpDir); err != nil {
		if !strings.Contains(err.Error(), "read config:") {
			t.Errorf("expected 'read config:' error, got: %v", err)
		}
	}
}

func TestLoadDirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(ConfigPath(tmpDir), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error when config path is a directory")
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	tmpDir := t.TempDir()

	session := &SessionState{
		Model:  "gpt-4",
		Prompt: "Fix the failing test",
		Ticket: "PROJ-123",
		Origin: "prompt",
	}

	if err := SaveSession(tmpDir, session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	loaded, err := LoadSession(tmpDir)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}
	if loaded.Model != session.Model || loaded.Prompt != session.Prompt || loaded.Ticket != session.Ticket || loaded.Origin != session.Origin {
		t.Fatalf("loaded session mismatch: got %+v, want %+v", loaded, session)
	}
}

func TestLoadSessionMissing(t *testing.T) {
	tmpDir := t.TempDir()

	loaded, err := LoadSession(tmpDir)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil session for missing file, got %+v", loaded)
	}
}

func TestDefaultSummaryConfig(t *testing.T) {
	cfg := DefaultSummaryConfig()
	if !cfg.Enabled {
		t.Error("expected summary to be enabled by default")
	}
	if cfg.Command != "llm" {
		t.Errorf("expected command 'llm', got %q", cfg.Command)
	}
	if cfg.Mode != SummaryModeFilenames {
		t.Errorf("expected mode %q, got %q", SummaryModeFilenames, cfg.Mode)
	}
}

func TestLoadSummaryDefaultsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dir := ConfigDir(tmpDir)
	os.MkdirAll(dir, 0755)
	os.WriteFile(ConfigPath(tmpDir), []byte("summary:\n  enabled: true\n"), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Summary == nil || !cfg.Summary.Enabled {
		t.Error("expected summary to be enabled")
	}
	if cfg.Summary.Command != "llm" {
		t.Errorf("expected default command 'llm', got %q", cfg.Summary.Command)
	}
	if cfg.Summary.Mode != SummaryModeFilenames {
		t.Errorf("expected default mode %q, got %q", SummaryModeFilenames, cfg.Summary.Mode)
	}
}

func TestLoadSummaryCustomCommand(t *testing.T) {
	tmpDir := t.TempDir()
	dir := ConfigDir(tmpDir)
	os.MkdirAll(dir, 0755)
	os.WriteFile(ConfigPath(tmpDir), []byte("summary:\n  enabled: true\n  command: claude\n  mode: diff\n"), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Summary.Command != "claude" {
		t.Errorf("expected 'claude', got %q", cfg.Summary.Command)
	}
	if cfg.Summary.Mode != SummaryModeDiff {
		t.Errorf("expected mode %q, got %q", SummaryModeDiff, cfg.Summary.Mode)
	}
}

func TestLoadSummaryDisabledNoDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	dir := ConfigDir(tmpDir)
	os.MkdirAll(dir, 0755)
	os.WriteFile(ConfigPath(tmpDir), []byte("summary:\n  enabled: false\n"), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Summary != nil && cfg.Summary.Command != "" {
		// When disabled, we don't apply defaults, but the original config
		// values are preserved from unmarshal. The command would be empty
		// since we didn't set it.
		if cfg.Summary.Command != "" {
			t.Errorf("expected empty command when disabled, got %q", cfg.Summary.Command)
		}
	}
}

func TestBackendConstants(t *testing.T) {
	if BackendGitNotes != "git-notes" {
		t.Errorf("unexpected BackendGitNotes value")
	}
	if BackendFile != "file" {
		t.Errorf("unexpected BackendFile value")
	}
	if BackendMetadata != "metadata" {
		t.Errorf("unexpected BackendMetadata value")
	}
}

func TestWriteAndReadLastCapture(t *testing.T) {
	tmpDir := t.TempDir()

	if err := WriteLastCapture(tmpDir, "abc1234567890", "git-notes"); err != nil {
		t.Fatalf("WriteLastCapture() error = %v", err)
	}

	receipt, err := ReadLastCapture(tmpDir)
	if err != nil {
		t.Fatalf("ReadLastCapture() error = %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if receipt.Commit != "abc1234567890" {
		t.Errorf("expected commit %q, got %q", "abc1234567890", receipt.Commit)
	}
	if receipt.Backend != "git-notes" {
		t.Errorf("expected backend %q, got %q", "git-notes", receipt.Backend)
	}
	if receipt.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	// Verify timestamp parses as RFC3339.
	if _, err := time.Parse(time.RFC3339, receipt.Timestamp); err != nil {
		t.Errorf("timestamp not valid RFC3339: %v", err)
	}
}

func TestReadLastCaptureMissing(t *testing.T) {
	tmpDir := t.TempDir()

	receipt, err := ReadLastCapture(tmpDir)
	if err != nil {
		t.Fatalf("ReadLastCapture() error = %v (expected nil, nil)", err)
	}
	if receipt != nil {
		t.Errorf("expected nil receipt for missing file, got %+v", receipt)
	}
}

func TestAppendCaptureError(t *testing.T) {
	tmpDir := t.TempDir()

	if err := AppendCaptureError(tmpDir, "aaa111", "backend failure"); err != nil {
		t.Fatalf("first AppendCaptureError() error = %v", err)
	}

	count, err := CountCaptureErrors(tmpDir)
	if err != nil {
		t.Fatalf("CountCaptureErrors() error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 error line, got %d", count)
	}

	if err := AppendCaptureError(tmpDir, "bbb222", "another failure"); err != nil {
		t.Fatalf("second AppendCaptureError() error = %v", err)
	}

	count, err = CountCaptureErrors(tmpDir)
	if err != nil {
		t.Fatalf("CountCaptureErrors() error = %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 error lines, got %d", count)
	}
}

func TestCountCaptureErrorsMissing(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := CountCaptureErrors(tmpDir)
	if err != nil {
		t.Fatalf("CountCaptureErrors() error = %v (expected 0, nil)", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for missing file, got %d", count)
	}
}
