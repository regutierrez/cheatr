package backend

import (
	"cheatr/internal/backend/parsers"
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented")

type service struct {
	sources  *SourceManager
	resolver Resolver
}

func New(dataDir string) (Backend, error) {
	sources, err := NewSourceManager(dataDir)
	if err != nil {
		return nil, err
	}

	resolver := newRoutingResolver(sources, sources)

	return &service{sources: sources, resolver: resolver}, nil
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
	if s.resolver == nil {
		return nil, notImplemented("Resolve")
	}

	return s.resolver.Resolve(args)
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

func (s *service) KnownLanguages() ([]string, error) {
	return s.sources.KnownLanguages()
}

func (s *service) IsLanguage(name string) bool {
	return s.sources.IsLanguage(name)
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
