package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName     = ".gitwhy"
	ConfigFileName    = "config.yaml"
	SessionFileName   = "session.yaml"
	DefaultBackend    = "git-notes"
	LastCaptureFile   = "last-capture"
	CaptureErrorsFile = "capture-errors.log"
	GitignoreFile     = ".gitignore"
)

// LastCapturePath returns the path to the last-capture receipt file.
func LastCapturePath(repoPath string) string {
	return filepath.Join(ConfigDir(repoPath), LastCaptureFile)
}

// CaptureErrorsPath returns the path to the capture errors log file.
func CaptureErrorsPath(repoPath string) string {
	return filepath.Join(ConfigDir(repoPath), CaptureErrorsFile)
}

// CaptureReceipt records the outcome of the most recent successful provenance capture.
type CaptureReceipt struct {
	Timestamp string `json:"timestamp"`
	Commit    string `json:"commit"`
	Backend   string `json:"backend"`
}

// WriteLastCapture writes a CaptureReceipt JSON file to LastCapturePath,
// creating parent directories as needed.
func WriteLastCapture(repoPath, commit, backend string) error {
	dir := ConfigDir(repoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	receipt := CaptureReceipt{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Commit:    commit,
		Backend:   backend,
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}
	if err := os.WriteFile(LastCapturePath(repoPath), data, 0644); err != nil {
		return fmt.Errorf("write last-capture: %w", err)
	}
	return nil
}

// ReadLastCapture reads and parses the last-capture receipt.
// Returns nil, nil if the file does not exist.
func ReadLastCapture(repoPath string) (*CaptureReceipt, error) {
	data, err := os.ReadFile(LastCapturePath(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read last-capture: %w", err)
	}
	var receipt CaptureReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, fmt.Errorf("parse last-capture: %w", err)
	}
	return &receipt, nil
}

// AppendCaptureError appends a timestamped error line to CaptureErrorsPath,
// creating the file (and parent dirs) as needed.
func AppendCaptureError(repoPath, commit, message string) error {
	dir := ConfigDir(repoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	f, err := os.OpenFile(CaptureErrorsPath(repoPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open capture-errors: %w", err)
	}
	defer f.Close()
	line := fmt.Sprintf("%s commit=%s: %s\n", time.Now().UTC().Format(time.RFC3339), commit, message)
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write capture error: %w", err)
	}
	return nil
}

// CountCaptureErrors counts the number of lines in CaptureErrorsPath.
// Returns 0, nil if the file does not exist.
func CountCaptureErrors(repoPath string) (int, error) {
	f, err := os.Open(CaptureErrorsPath(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("open capture-errors: %w", err)
	}
	defer f.Close()
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan capture-errors: %w", err)
	}
	return count, nil
}

type StorageBackend string

const (
	BackendGitNotes StorageBackend = "git-notes"
	BackendFile     StorageBackend = "file"
	BackendMetadata StorageBackend = "metadata"
)

type Config struct {
	Backend      StorageBackend `yaml:"backend"`
	RepoPath     string         `yaml:"-"`
	DefaultModel string         `yaml:"default_model,omitempty"`

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
