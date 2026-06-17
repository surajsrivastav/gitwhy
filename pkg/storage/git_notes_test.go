package storage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return tmpDir
}

func commitFile(t *testing.T, repoPath, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmds := [][]string{
		{"add", filename},
		{"commit", "-m", "add " + filename},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestGitNotesBackendName(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)
	if backend.Name() != "git-notes" {
		t.Errorf("expected 'git-notes', got %q", backend.Name())
	}
}

func TestGitNotesBackendStoreAndGet(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "test.txt", "hello")
	backend := NewGitNotesBackend(repoPath)

	record := provenance.NewRecord(provenance.TargetCommit, commitHash)
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("test intent", provenance.OriginSpec, "SPEC-1", "hash123")

	if err := backend.Store(record); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := backend.Get(commitHash)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Target.Ref != commitHash {
		t.Errorf("ref mismatch: %q", got.Target.Ref)
	}
	if got.Attribution.By != "agent:blue" {
		t.Errorf("attribution mismatch: %q", got.Attribution.By)
	}
	if got.Intent.Spec != "SPEC-1" {
		t.Errorf("spec mismatch: %q", got.Intent.Spec)
	}
}

func TestGitNotesBackendGetNonExistent(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "test.txt", "hello")
	backend := NewGitNotesBackend(repoPath)

	_, err := backend.Get(commitHash)
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
	if err != nil && !strings.Contains(err.Error(), "no provenance record for "+commitHash) {
		t.Errorf("expected 'no provenance record' error, got: %v", err)
	}
}

func TestGitNotesBackendGetWithEmptyRef(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)

	_, err := backend.Get("")
	if err == nil {
		t.Error("expected error when ref is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "git notes show") {
		t.Errorf("expected 'git notes show' error, got: %v", err)
	}
}

func TestGitNotesBackendList(t *testing.T) {
	repoPath := initGitRepo(t)

	commit1 := commitFile(t, repoPath, "a.txt", "a")
	commit2 := commitFile(t, repoPath, "b.txt", "b")

	backend := NewGitNotesBackend(repoPath)

	r1 := provenance.NewRecord(provenance.TargetCommit, commit1)
	r1.SetIntent("first", provenance.OriginHuman, "", "")
	if err := backend.Store(r1); err != nil {
		t.Fatal(err)
	}

	r2 := provenance.NewRecord(provenance.TargetCommit, commit2)
	r2.SetIntent("second", provenance.OriginHuman, "", "")
	if err := backend.Store(r2); err != nil {
		t.Fatal(err)
	}

	records, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	refs := map[string]bool{commit1: true, commit2: true}
	for _, r := range records {
		if !refs[r.Target.Ref] {
			t.Errorf("unexpected ref %q in list results", r.Target.Ref)
		}
	}
}

func TestGitNotesBackendListEmpty(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)

	records, err := backend.List()
	if err != nil {
		t.Fatalf("List() on empty should not error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty list, got %d records", len(records))
	}
}

func TestGitNotesBackendListNonGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewGitNotesBackend(tmpDir)

	_, err := backend.List()
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestGitNotesBackendClose(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)
	if err := backend.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestGitNotesBackendOverwrite(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "test.txt", "hello")
	backend := NewGitNotesBackend(repoPath)

	r1 := provenance.NewRecord(provenance.TargetCommit, commitHash)
	r1.SetIntent("first", provenance.OriginHuman, "", "")
	if err := backend.Store(r1); err != nil {
		t.Fatal(err)
	}

	r2 := provenance.NewRecord(provenance.TargetCommit, commitHash)
	r2.SetIntent("second", provenance.OriginSpec, "SPEC-2", "hash2")
	if err := backend.Store(r2); err != nil {
		t.Fatal(err)
	}

	got, err := backend.Get(commitHash)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent.Summary != "second" {
		t.Errorf("expected 'second', got %q", got.Intent.Summary)
	}
}

func TestGitNotesBackendMarshalJSON(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)

	data, err := backend.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
	if !strings.Contains(string(data), "git-notes") {
		t.Error("expected JSON to contain git-notes type")
	}
}

func TestGitNotesBackendStoreWithPR(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "test.txt", "hello")
	backend := NewGitNotesBackend(repoPath)

	record := provenance.NewRecord(provenance.TargetPR, commitHash)
	record.SetAttribution(provenance.AttributionHuman)
	record.SetIntent("pr intent", provenance.OriginHuman, "", "")

	err := backend.Store(record)
	if err != nil {
		t.Fatalf("Store() with PR target should not error: %v", err)
	}

	got, err := backend.Get(commitHash)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Target.Type != provenance.TargetPR {
		t.Errorf("expected PR target type, got %q", got.Target.Type)
	}
}

func TestGitNotesBackendListWithInvalidNotes(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "a.txt", "hello")
	commitHash2 := commitFile(t, repoPath, "b.txt", "world")
	backend := NewGitNotesBackend(repoPath)

	record := provenance.NewRecord(provenance.TargetCommit, commitHash)
	record.SetIntent("valid", provenance.OriginHuman, "", "")
	if err := backend.Store(record); err != nil {
		t.Fatal(err)
	}

	invalidNoteCmd := exec.Command("git", "-C", repoPath, "notes", "--ref", NotesRef, "add", "-f", "-m", "not-json", commitHash2)
	if out, err := invalidNoteCmd.CombinedOutput(); err != nil {
		t.Fatalf("creating invalid note: %v\n%s", err, out)
	}

	records, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 valid record (invalid note skipped), got %d", len(records))
	}
}

func TestGitNotesBackendListBreakOnInvalidNote(t *testing.T) {
	repoPath := initGitRepo(t)
	commit1 := commitFile(t, repoPath, "a.txt", "first")
	commit2 := commitFile(t, repoPath, "b.txt", "second")

	backend := NewGitNotesBackend(repoPath)

	r1 := provenance.NewRecord(provenance.TargetCommit, commit1)
	r1.SetIntent("valid-1", provenance.OriginHuman, "", "")
	if err := backend.Store(r1); err != nil {
		t.Fatal(err)
	}

	r2 := provenance.NewRecord(provenance.TargetCommit, commit2)
	r2.SetIntent("valid-2", provenance.OriginHuman, "", "")
	if err := backend.Store(r2); err != nil {
		t.Fatal(err)
	}

	invalidNoteCmd := exec.Command("git", "-C", repoPath, "notes", "--ref", NotesRef, "add", "-f", "-m", `{{invalid json}`, commit2)
	if out, err := invalidNoteCmd.CombinedOutput(); err != nil {
		t.Fatalf("overwriting note with invalid JSON: %v\n%s", err, out)
	}

	records, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) == 0 {
		t.Error("expected at least the valid note (commit1) to be returned, got 0")
	}
}

func TestGitNotesBackendStoreWithInvalidRef(t *testing.T) {
	repoPath := initGitRepo(t)
	backend := NewGitNotesBackend(repoPath)

	record := provenance.NewRecord(provenance.TargetCommit, "invalid-ref-that-does-not-exist")
	record.SetIntent("test", provenance.OriginHuman, "", "")

	err := backend.Store(record)
	if err == nil {
		t.Error("expected error when storing with invalid ref")
	}
}

func TestGitNotesBackendGetWithInvalidJSON(t *testing.T) {
	repoPath := initGitRepo(t)
	commitHash := commitFile(t, repoPath, "test.txt", "hello")
	backend := NewGitNotesBackend(repoPath)

	invalidNoteCmd := exec.Command("git", "-C", repoPath, "notes", "--ref", NotesRef, "add", "-f", "-m", "not-json", commitHash)
	if out, err := invalidNoteCmd.CombinedOutput(); err != nil {
		t.Fatalf("creating invalid note: %v\n%s", err, out)
	}

	_, err := backend.Get(commitHash)
	if err == nil {
		t.Error("expected error when note contains invalid JSON")
	}
}

func TestEnsureNotesRef(t *testing.T) {
	repoPath := initGitRepo(t)
	commitFile(t, repoPath, "test.txt", "hello")

	if err := EnsureNotesRef(repoPath); err != nil {
		t.Fatalf("EnsureNotesRef() error = %v", err)
	}
}

func TestFactoryGetGitNotesBackend(t *testing.T) {
	repoPath := initGitRepo(t)
	f := NewFactory()
	f.Register("git-notes", NewGitNotesBackend(repoPath))

	got, err := f.Get("git-notes")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "git-notes" {
		t.Errorf("expected name 'git-notes', got %q", got.Name())
	}
}
