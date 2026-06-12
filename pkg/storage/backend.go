package storage

import (
	"fmt"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

type Backend interface {
	Name() string
	Store(record *provenance.Record) error
	Get(ref string) (*provenance.Record, error)
	List() ([]*provenance.Record, error)
	Close() error
}

type Factory struct {
	backends map[string]Backend
}

func NewFactory() *Factory {
	return &Factory{
		backends: make(map[string]Backend),
	}
}

func (f *Factory) Register(name string, backend Backend) {
	f.backends[name] = backend
}

func (f *Factory) Get(name string) (Backend, error) {
	backend, ok := f.backends[name]
	if !ok {
		return nil, fmt.Errorf("unknown storage backend: %s", name)
	}
	return backend, nil
}

func (f *Factory) Available() []string {
	var names []string
	for name := range f.backends {
		names = append(names, name)
	}
	return names
}
