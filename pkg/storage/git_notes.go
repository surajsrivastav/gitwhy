package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
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

// isGitObject reports whether ref resolves to an existing git object.
func isGitObject(repoPath, ref string) bool {
	if ref == "" {
		return false
	}
	cmd := exec.Command("git", "-C", repoPath, "cat-file", "-e", ref)
	return cmd.Run() == nil
}

// isPRNumber reports whether s looks like a bare PR number ("123").
func isPRNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// prBlobSHA returns the git blob object ID for the synthetic anchor content
// used for bare PR-number targets. Git blob IDs are content-addressed as
// sha1("blob <len>\0" + content), so both Store and Get can recompute the
// anchor deterministically without storing a mapping anywhere.
func prBlobSHA(prNumber string) string {
	content := "gitwhy-pr:" + prNumber
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00%s", len(content), content)
	return hex.EncodeToString(h.Sum(nil))
}

// noteTarget resolves the git object a record's note should be attached to.
// PR records carry a bare PR number rather than a git object, which git notes
// cannot attach to — so they are anchored to a synthetic blob whose content is
// derived from the PR number (written into the object store via hash-object).
func (g *GitNotesBackend) noteTarget(record *provenance.Record) (string, error) {
	if record.Target.Type == provenance.TargetPR && !isGitObject(g.repoPath, record.Target.Ref) {
		blobSHA := prBlobSHA(record.Target.Ref)
		cmd := exec.Command("git", "-C", g.repoPath, "hash-object", "-w", "--stdin")
		cmd.Stdin = strings.NewReader("gitwhy-pr:" + record.Target.Ref)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git hash-object: %w\noutput: %s", err, string(output))
		}
		// Sanity check: the object git wrote must be the deterministic anchor.
		if written := strings.TrimSpace(string(output)); written != blobSHA {
			return "", fmt.Errorf("hash-object wrote %s, expected anchor %s", written, blobSHA)
		}
		return blobSHA, nil
	}
	return record.Target.Ref, nil
}

func (g *GitNotesBackend) Store(record *provenance.Record) error {
	data, err := record.Marshal()
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	target, err := g.noteTarget(record)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "-C", g.repoPath,
		"notes", "--ref", NotesRef, "add", "-f", "-m", string(data), target,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git notes add: %w\noutput: %s", err, string(output))
	}
	return nil
}

func (g *GitNotesBackend) Get(ref string) (*provenance.Record, error) {
	// Bare PR numbers are anchored to synthetic blobs; resolve to the blob SHA.
	target := ref
	if isPRNumber(ref) && !isGitObject(g.repoPath, ref) {
		target = prBlobSHA(ref)
	}
	cmd := exec.Command("git", "-C", g.repoPath,
		"notes", "--ref", NotesRef, "show", target,
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
		// Only fall back to the annotated object when the record itself
		// carries no ref — PR records anchored to synthetic blobs must keep
		// their PR number, not the blob SHA.
		if record.Target.Ref == "" {
			record.Target.Ref = commitRef
		}
		records = append(records, record)
	}

	return records, nil
}

func (g *GitNotesBackend) Close() error {
	return nil
}

func EnsureNotesRef(repoPath string) error {
	// If HEAD already has a note, leave it untouched. The previous
	// implementation added a placeholder (which fails when a note exists)
	// and then removed HEAD's note unconditionally — destroying the real
	// provenance record.
	if err := exec.Command("git", "-C", repoPath,
		"notes", "--ref", NotesRef, "show", "HEAD",
	).Run(); err == nil {
		return nil
	}

	// No note on HEAD: create the notes ref with a placeholder, then remove
	// only the placeholder this function just created.
	if err := exec.Command("git", "-C", repoPath,
		"notes", "--ref", NotesRef, "add", "-m", "{}", "HEAD",
	).Run(); err != nil {
		return fmt.Errorf("create notes ref: %w", err)
	}
	if err := exec.Command("git", "-C", repoPath,
		"notes", "--ref", NotesRef, "remove", "HEAD",
	).Run(); err != nil {
		return fmt.Errorf("remove placeholder note: %w", err)
	}
	return nil
}

var _ Backend = (*GitNotesBackend)(nil)

func (g *GitNotesBackend) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"type": "git-notes"})
}
