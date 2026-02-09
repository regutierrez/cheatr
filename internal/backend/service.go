package backend

import (
	"cheatr/internal/backend/parsers"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrNotImplemented = errors.New("not implemented")

type service struct {
	sources  *SourceManager
	resolver Resolver
	renderer contentRenderer
}

func New(dataDir string) (Backend, error) {
	sources, err := NewSourceManager(dataDir)
	if err != nil {
		return nil, err
	}

	resolver := newRoutingResolver(sources, sources)

	return &service{sources: sources, resolver: resolver, renderer: newMarkdownRenderer()}, nil
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
	if res == nil {
		return nil, errors.New("resolution is required")
	}

	entry, err := s.entryFromResolution(res)
	if err != nil {
		return nil, err
	}

	markdown, err := s.renderer.RenderEntry(entry)
	if err != nil {
		return nil, err
	}

	rendered := *entry
	rendered.Content = markdown
	return &rendered, nil
}

func (s *service) GetSection(res *Resolution) (string, error) {
	if res == nil {
		return "", errors.New("resolution is required")
	}

	if res.Subtopic == "" {
		entry, err := s.GetEntry(res)
		if err != nil {
			return "", err
		}
		return s.renderer.RenderEntry(entry)
	}

	entry, err := s.entryFromResolution(res)
	if err == nil {
		if markdown, sectionErr := s.renderer.RenderSection(entry, res.Subtopic); sectionErr == nil {
			return markdown, nil
		}
	}

	if res.Content == "" {
		return "", ErrResolutionNotFound
	}

	return s.renderer.RenderMarkdown(res.Content)
}

func (s *service) Search(query string, filter SourceFilter) ([]SearchResult, error) {
	if s.resolver == nil {
		return nil, notImplemented("Search")
	}

	return s.resolver.Search(query, filter)
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

func (s *service) entryFromResolution(res *Resolution) (*parsers.Entry, error) {
	if res == nil {
		return nil, errors.New("resolution is required")
	}

	source := normalizeTopic(res.Source)
	topic := normalizeTopic(res.Topic)

	if source == "" {
		if strings.TrimSpace(res.Content) == "" {
			return nil, ErrResolutionNotFound
		}
		return &parsers.Entry{
			Source:  res.Source,
			Topic:   res.Topic,
			Title:   res.Topic,
			Content: res.Content,
		}, nil
	}

	switch source {
	case SourceLXIYM, SourceDevhints, SourceTldr:
		if topic == "" {
			return nil, errors.New("resolution topic is required")
		}

		repoPath, err := s.sources.RepoPath(source)
		if err != nil {
			return nil, err
		}

		entries, err := loadEntriesBySource(context.Background(), source, repoPath, s.sources.cache)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if normalizeTopic(entry.Topic) != topic {
				continue
			}

			if source != SourceTldr || strings.Contains(entry.ID, ":common:") {
				return entry, nil
			}
		}

		for _, entry := range entries {
			if normalizeTopic(entry.Topic) == topic {
				return entry, nil
			}
		}

		return nil, ErrResolutionNotFound
	case SourceDevDocs, SourceLLM:
		if strings.TrimSpace(res.Content) == "" {
			return nil, ErrResolutionNotFound
		}

		title := topic
		if title == "" {
			title = normalizeTopic(res.Subtopic)
		}
		if title == "" {
			title = source
		}

		return &parsers.Entry{
			Source:  source,
			Topic:   topic,
			Title:   title,
			Content: res.Content,
		}, nil
	default:
		if strings.TrimSpace(res.Content) == "" {
			return nil, ErrResolutionNotFound
		}

		return &parsers.Entry{
			Source:  source,
			Topic:   topic,
			Title:   topic,
			Content: res.Content,
		}, nil
	}
}
