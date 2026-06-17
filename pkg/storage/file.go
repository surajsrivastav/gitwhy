package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

type FileBackend struct {
	storageDir string
}

func NewFileBackend(storageDir string) *FileBackend {
	return &FileBackend{storageDir: storageDir}
}

func (f *FileBackend) Name() string {
	return "file"
}

func (f *FileBackend) recordPath(ref string) string {
	safe := strings.ReplaceAll(ref, "/", "_")
	return filepath.Join(f.storageDir, fmt.Sprintf("%s.json", safe))
}

func (f *FileBackend) Store(record *provenance.Record) error {
	if err := os.MkdirAll(f.storageDir, 0755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	data, err := record.Marshal()
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	if err := os.WriteFile(f.recordPath(record.Target.Ref), data, 0644); err != nil {
		return fmt.Errorf("write record file: %w", err)
	}
	return nil
}

func (f *FileBackend) Get(ref string) (*provenance.Record, error) {
	data, err := os.ReadFile(f.recordPath(ref))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no provenance record for %s", ref)
		}
		return nil, fmt.Errorf("read record: %w", err)
	}

	return provenance.Unmarshal(data)
}

func (f *FileBackend) List() ([]*provenance.Record, error) {
	entries, err := os.ReadDir(f.storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read storage dir: %w", err)
	}

	var records []*provenance.Record
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(f.storageDir, entry.Name()))
		if err != nil {
			continue
		}

		record, err := provenance.Unmarshal(data)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

func (f *FileBackend) Close() error {
	return nil
}

var _ Backend = (*FileBackend)(nil)

func (f *FileBackend) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"type": "file", "path": f.storageDir})
}
