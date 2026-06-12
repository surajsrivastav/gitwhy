package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anomalyco/gitwhy/pkg/provenance"
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

	os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte("not json"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "notjson.txt"), []byte("ignore me"), 0644)

	record := provenance.NewRecord(provenance.TargetCommit, "good")
	backend.Store(record)

	list, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 valid record, got %d", len(list))
	}
}
