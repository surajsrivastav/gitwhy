package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

func TestFileBackendStoreAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")

	if err := backend.Store(record); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := backend.Get("abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Target.Ref != "abc123" {
		t.Errorf("ref mismatch: %q", got.Target.Ref)
	}
	if got.Attribution.By != "agent:blue" {
		t.Errorf("attribution mismatch: %q", got.Attribution.By)
	}
	if got.Intent.Spec != "BLUE-317" {
		t.Errorf("spec mismatch: %q", got.Intent.Spec)
	}
}

func TestFileBackendGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	_, err := backend.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent record")
	}
	if err != nil && !strings.Contains(err.Error(), "no provenance record for nonexistent") {
		t.Errorf("expected 'no provenance record' error, got: %v", err)
	}
}

func TestFileBackendList(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	records := []*provenance.Record{
		provenance.NewRecord(provenance.TargetCommit, "aaa"),
		provenance.NewRecord(provenance.TargetCommit, "bbb"),
		provenance.NewRecord(provenance.TargetPR, "1"),
	}

	for _, r := range records {
		if err := backend.Store(r); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 records, got %d", len(list))
	}
}

func TestFileBackendListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() on empty backend error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d records", len(list))
	}
}

func TestFileBackendName(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	if backend.Name() != "file" {
		t.Errorf("expected 'file', got %q", backend.Name())
	}
}

func TestFileBackendOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	r1 := provenance.NewRecord(provenance.TargetCommit, "abc")
	r1.SetIntent("first", provenance.OriginHuman, "", "")

	r2 := provenance.NewRecord(provenance.TargetCommit, "abc")
	r2.SetIntent("second", provenance.OriginSpec, "BLUE-1", "hash1")

	if err := backend.Store(r1); err != nil {
		t.Fatal(err)
	}
	if err := backend.Store(r2); err != nil {
		t.Fatal(err)
	}

	got, err := backend.Get("abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent.Summary != "second" {
		t.Errorf("expected 'second', got %q", got.Intent.Summary)
	}
}

func TestFileBackendMarshalsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	data, err := backend.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestFileBackendClose(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	if err := backend.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestFileBackendStorageDirCreation(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nested", "dir")
	backend := NewFileBackend(tmpDir)

	record := provenance.NewRecord(provenance.TargetCommit, "abc")
	if err := backend.Store(record); err != nil {
		t.Fatalf("Store() with nested dir error = %v", err)
	}

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("storage directory was not created")
	}
}

func TestFileBackendInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	validNonJSON := `{"target":{"type":"commit","ref":"nonjson"},"attribution":{"by":"agent:test"},"intent":{"summary":"test","origin":"spec","spec":"TEST-1","hash":"abc"},"timestamp":"0001-01-01T00:00:00Z"}`
	os.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte(validNonJSON), 0644)
	os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte("not json"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "notjson.txt"), []byte(validNonJSON), 0644)

	record := provenance.NewRecord(provenance.TargetCommit, "good")
	backend.Store(record)

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 valid records (skipping bad.json), got %d", len(list))
	}
}

func TestFileBackendListContinueVsBreak(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	r1 := provenance.NewRecord(provenance.TargetCommit, "aaa")
	backend.Store(r1)

	r2 := provenance.NewRecord(provenance.TargetCommit, "ccc")
	backend.Store(r2)

	nonJSONPath := filepath.Join(tmpDir, "bbb.txt")
	os.WriteFile(nonJSONPath, []byte("not json"), 0644)

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 records (non-json file should be skipped), got %d", len(list))
	}
}

func TestFileBackendRecordPathSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	record := provenance.NewRecord(provenance.TargetCommit, "feature/branch/ref")
	if err := backend.Store(record); err != nil {
		t.Fatalf("Store() with slashes in ref error = %v", err)
	}

	got, err := backend.Get("feature/branch/ref")
	if err != nil {
		t.Fatalf("Get() with slashes in ref error = %v", err)
	}
	if got.Target.Ref != "feature/branch/ref" {
		t.Errorf("ref mismatch: %q", got.Target.Ref)
	}
}

func TestFileBackendGetReadError(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	record := provenance.NewRecord(provenance.TargetCommit, "badref")
	record.SetIntent("test", provenance.OriginHuman, "", "")
	if err := backend.Store(record); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(tmpDir, "badref.json")
	if err := os.Chmod(badPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(badPath, 0644) })

	_, err := backend.Get("badref")
	if err == nil {
		t.Error("expected error when file can't be read")
	}
	if err != nil && !strings.Contains(err.Error(), "read record") {
		t.Errorf("expected 'read record' error, got: %v", err)
	}
}

func TestFileBackendStoreWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "subdir")

	if err := os.WriteFile(storagePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	backend := NewFileBackend(storagePath)
	record := provenance.NewRecord(provenance.TargetCommit, "test")
	err := backend.Store(record)
	if err == nil {
		t.Error("expected error when storage path is a file")
	}
	if err != nil && !strings.Contains(err.Error(), "create storage dir") {
		t.Errorf("expected 'create storage dir' error, got: %v", err)
	}
}

func TestFileBackendStoreWriteFileError(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}

	backend := NewFileBackend(readOnlyDir)
	record := provenance.NewRecord(provenance.TargetCommit, "test")
	err := backend.Store(record)
	if err == nil {
		t.Error("expected error when storage dir is read-only")
	}
}

func TestFileBackendListNonExistentDir(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nonexistent")
	backend := NewFileBackend(tmpDir)

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() on non-existent dir error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestFileBackendListReadDirError(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(storagePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	backend := NewFileBackend(storagePath)
	_, err := backend.List()
	if err == nil {
		t.Error("expected error when storage path is a file")
	}
}

func TestFileBackendListReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)

	badPath := filepath.Join(tmpDir, "aaa-unreadable.json")
	if err := os.WriteFile(badPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(badPath, 0644) })

	record := provenance.NewRecord(provenance.TargetCommit, "zzz-good")
	if err := backend.Store(record); err != nil {
		t.Fatal(err)
	}

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 valid record (unreadable file skipped), got %d", len(list))
	}
}

func TestFileBackendMarshalJSONContent(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	data, err := backend.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"path":"`+tmpDir+`","type":"file"}` {
		t.Errorf("unexpected JSON: %s", data)
	}
}
