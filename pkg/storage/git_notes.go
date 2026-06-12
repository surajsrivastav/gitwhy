package storage

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

const NotesRef = "refs/notes/gitwhy"

type GitNotesBackend struct {
	repoPath string
}

func NewGitNotesBackend(repoPath string) *GitNotesBackend {
	return &GitNotesBackend{repoPath: repoPath}
}

func (g *GitNotesBackend) Name() string {
	return "git-notes"
}

func (g *GitNotesBackend) Store(record *provenance.Record) error {
	data, err := record.Marshal()
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	cmd := exec.Command("git", "-C", g.repoPath,
		"notes", "--ref", NotesRef, "add", "-f", "-m", string(data), record.Target.Ref,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git notes add: %w\noutput: %s", err, string(output))
	}
	return nil
}

func (g *GitNotesBackend) Get(ref string) (*provenance.Record, error) {
	cmd := exec.Command("git", "-C", g.repoPath,
		"notes", "--ref", NotesRef, "show", ref,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no note found") {
			return nil, fmt.Errorf("no provenance record for %s", ref)
		}
		return nil, fmt.Errorf("git notes show: %w\noutput: %s", err, string(output))
	}

	record, err := provenance.Unmarshal(output)
	if err != nil {
		return nil, fmt.Errorf("unmarshal record: %w", err)
	}
	return record, nil
}

func (g *GitNotesBackend) List() ([]*provenance.Record, error) {
	cmd := exec.Command("git", "-C", g.repoPath,
		"notes", "--ref", NotesRef, "list",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git notes list: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var records []*provenance.Record

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		_, commitRef := parts[0], parts[1]

		showCmd := exec.Command("git", "-C", g.repoPath, "notes", "--ref", NotesRef, "show", commitRef)
		noteOutput, err := showCmd.CombinedOutput()
		if err != nil {
			continue
		}

		record, err := provenance.Unmarshal(noteOutput)
		if err != nil {
			continue
		}
		record.Target.Ref = commitRef
		records = append(records, record)
	}

	return records, nil
}

func (g *GitNotesBackend) Close() error {
	return nil
}

func EnsureNotesRef(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath,
		"notes", "--ref", NotesRef, "add", "-m", "{}", "HEAD",
	)
	cmd.Run()

	cmd = exec.Command("git", "-C", repoPath,
		"notes", "--ref", NotesRef, "remove", "HEAD",
	)
	cmd.Run()
	return nil
}

var _ Backend = (*GitNotesBackend)(nil)

func (g *GitNotesBackend) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"type": "git-notes"})
}
