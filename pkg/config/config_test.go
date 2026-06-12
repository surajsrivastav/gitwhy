package config

import (
	"os"
	"path/filepath"
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

	_, err := FindRepoRoot()
	if err == nil {
		t.Skip("not in a git repo — this is expected in non-git dirs")
	}

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
