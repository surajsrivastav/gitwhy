package storage

import (
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

type mockBackend struct {
	name    string
	records map[string]*provenance.Record
}

func (m *mockBackend) Name() string { return m.name }

func (m *mockBackend) Store(record *provenance.Record) error {
	if m.records == nil {
		m.records = make(map[string]*provenance.Record)
	}
	m.records[record.Target.Ref] = record
	return nil
}

func (m *mockBackend) Get(ref string) (*provenance.Record, error) {
	r, ok := m.records[ref]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockBackend) List() ([]*provenance.Record, error) {
	var list []*provenance.Record
	for _, r := range m.records {
		list = append(list, r)
	}
	return list, nil
}

func (m *mockBackend) Close() error { return nil }

func TestFactory(t *testing.T) {
	f := NewFactory()

	if len(f.Available()) != 0 {
		t.Error("expected no backends initially")
	}

	mock := &mockBackend{name: "test"}
	f.Register("test", mock)

	got, err := f.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "test" {
		t.Errorf("expected name 'test', got %q", got.Name())
	}

	avail := f.Available()
	if len(avail) != 1 || avail[0] != "test" {
		t.Errorf("unexpected available backends: %v", avail)
	}
}

func TestFactoryUnknownBackend(t *testing.T) {
	f := NewFactory()
	_, err := f.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown backend")
	}
}

func TestFactoryDuplicateRegistration(t *testing.T) {
	f := NewFactory()
	mock1 := &mockBackend{name: "dup"}
	mock2 := &mockBackend{name: "dup"}

	f.Register("dup", mock1)
	f.Register("dup", mock2)

	got, _ := f.Get("dup")
	if got.Name() != "dup" {
		t.Error("expected last registered backend")
	}
}
