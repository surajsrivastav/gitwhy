package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName   = ".gitwhy"
	ConfigFileName  = "config.yaml"
	SessionFileName = "session.yaml"
	DefaultBackend  = "git-notes"
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

	GitNotes    *GitNotesConfig    `yaml:"git_notes,omitempty"`
	File        *FileConfig        `yaml:"file,omitempty"`
	GitHub      *GitHubConfig      `yaml:"github,omitempty"`
	AutoCapture *AutoCaptureConfig `yaml:"auto_capture,omitempty"`
	Summary     *SummaryConfig     `yaml:"summary,omitempty"`
}

type SessionState struct {
	Model  string `yaml:"model,omitempty"`
	Prompt string `yaml:"prompt,omitempty"`
	Ticket string `yaml:"ticket,omitempty"`
	Origin string `yaml:"origin,omitempty"`
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

type AutoCaptureConfig struct {
	Enabled   bool   `yaml:"enabled"`
	DefaultBy string `yaml:"default_by,omitempty"`
}

type SummaryMode string

const (
	SummaryModeFilenames SummaryMode = "filenames"
	SummaryModeDiff      SummaryMode = "diff"
)

type SummaryConfig struct {
	Enabled bool        `yaml:"enabled"`
	Command string      `yaml:"command,omitempty"`
	Mode    SummaryMode `yaml:"mode,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Backend: BackendGitNotes,
		GitNotes: &GitNotesConfig{
			Enabled: true,
		},
	}
}

func DefaultSummaryConfig() *SummaryConfig {
	return &SummaryConfig{
		Enabled: true,
		Command: "llm",
		Mode:    SummaryModeFilenames,
	}
}

func ConfigDir(repoPath string) string {
	return filepath.Join(repoPath, ConfigDirName)
}

func ConfigPath(repoPath string) string {
	return filepath.Join(ConfigDir(repoPath), ConfigFileName)
}

func SessionPath(repoPath string) string {
	return filepath.Join(ConfigDir(repoPath), SessionFileName)
}

func LoadSession(repoPath string) (*SessionState, error) {
	path := SessionPath(repoPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var session SessionState
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	return &session, nil
}

func SaveSession(repoPath string, session *SessionState) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.Model == "" && session.Prompt == "" && session.Ticket == "" && session.Origin == "" {
		return fmt.Errorf("no session data to save")
	}

	dir := ConfigDir(repoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := os.WriteFile(SessionPath(repoPath), data, 0644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	return nil
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
		// Extract line number hint from yaml error if available.
		hint := ""
		if strings.Contains(err.Error(), "line ") {
			hint = " (" + err.Error() + ")"
		}
		fmt.Fprintf(os.Stderr, "  gitwhy: config invalid%s — using defaults\n", hint)
		return DefaultConfig(), nil
	}

	if cfg.Summary != nil && cfg.Summary.Enabled {
		defaults := DefaultSummaryConfig()
		if cfg.Summary.Command == "" {
			cfg.Summary.Command = defaults.Command
		}
		if cfg.Summary.Mode == "" {
			cfg.Summary.Mode = defaults.Mode
		}
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

	data, _ := yaml.Marshal(cfg)

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
