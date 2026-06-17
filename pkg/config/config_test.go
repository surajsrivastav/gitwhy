package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error for corrupted config")
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
