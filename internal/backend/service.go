package backend

import (
	"cheatr/internal/backend/parsers"
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented")

type service struct {
	sources *SourceManager
}

func New(dataDir string) (Backend, error) {
	sources, err := NewSourceManager(dataDir)
	if err != nil {
		return nil, err
	}

	return &service{sources: sources}, nil
}

func (s *service) Init() error {
	return s.sources.Init()
}

func (s *service) Update() error {
	return s.sources.Update()
}

func (s *service) UpdateSource(name string) error {
	return s.sources.UpdateSource(name)
}

func (s *service) Resolve(args []string) (*Resolution, error) {
	return nil, notImplemented("Resolve")
}

func (s *service) GetEntry(res *Resolution) (*parsers.Entry, error) {
	return nil, notImplemented("GetEntry")
}

func (s *service) GetSection(res *Resolution) (string, error) {
	return "", notImplemented("GetSection")
}

func (s *service) Search(query string, filter SourceFilter) ([]SearchResult, error) {
	return nil, notImplemented("Search")
}

func (s *service) ListDevDocs() ([]DevDoc, error) {
	return s.sources.ListDevDocs()
}

func (s *service) EnableDevDoc(slug string) error {
	return s.sources.EnableDevDoc(slug)
}

func (s *service) DisableDevDoc(slug string) error {
	return s.sources.DisableDevDoc(slug)
}

func notImplemented(method string) error {
	return fmt.Errorf("%s: %w", method, ErrNotImplemented)
}
