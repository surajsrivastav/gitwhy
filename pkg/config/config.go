package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName  = ".gitwhy"
	ConfigFileName = "config.yaml"
	DefaultBackend = "git-notes"
)

type StorageBackend string

const (
	BackendGitNotes StorageBackend = "git-notes"
	BackendFile     StorageBackend = "file"
	BackendMetadata StorageBackend = "metadata"
)

type Config struct {
	Backend  StorageBackend `yaml:"backend"`
	RepoPath string         `yaml:"-"`

	GitNotes *GitNotesConfig `yaml:"git_notes,omitempty"`
	File     *FileConfig     `yaml:"file,omitempty"`
	GitHub   *GitHubConfig   `yaml:"github,omitempty"`
}

type GitNotesConfig struct {
	Enabled bool `yaml:"enabled"`
}

type FileConfig struct {
	Path string `yaml:"path"`
}

type GitHubConfig struct {
	Token string `yaml:"token,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Backend: BackendGitNotes,
		GitNotes: &GitNotesConfig{
			Enabled: true,
		},
	}
}

func ConfigDir(repoPath string) string {
	return filepath.Join(repoPath, ConfigDirName)
}

func ConfigPath(repoPath string) string {
	return filepath.Join(ConfigDir(repoPath), ConfigFileName)
}

func Load(repoPath string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.RepoPath = repoPath

	data, err := os.ReadFile(ConfigPath(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	if cfg.RepoPath == "" {
		return fmt.Errorf("repo path not set")
	}

	dir := ConfigDir(cfg.RepoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(cfg.RepoPath), data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func IsInitialized(repoPath string) bool {
	_, err := os.Stat(ConfigPath(repoPath))
	return err == nil
}

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repository")
		}
		dir = parent
	}
}
