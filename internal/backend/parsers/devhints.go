package parsers

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type DevhintsLoader struct{}

func NewDevhintsLoader() *DevhintsLoader {
	return &DevhintsLoader{}
}

func (l *DevhintsLoader) Source() string {
	return SourceDevhints
}

func (l *DevhintsLoader) Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error) {
	return discoverMarkdownFiles(ctx, l.Source(), opts.Root, func(path string, _ fs.DirEntry) bool {
		base := strings.ToLower(filepath.Base(path))
		return base != "readme.md"
	}, opts.Limit)
}

func (l *DevhintsLoader) Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error) {
	return ParseDevhintsCandidate(ctx, candidate, opts)
}

func (l *DevhintsLoader) Load(ctx context.Context, opts LoadOptions) ([]*Entry, error) {
	candidates, err := l.Discover(ctx, opts.Discover)
	if err != nil {
		return nil, err
	}

	return loadWithParser(ctx, l.Source(), candidates, l.Parse, opts)
}

func LoadDevhintsEntries(ctx context.Context, repoRoot string) ([]*Entry, error) {
	loader := NewDevhintsLoader()
	return loader.Load(ctx, LoadOptions{Discover: DiscoverOptions{Root: repoRoot}})
}

func ParseDevhintsCandidate(_ context.Context, candidate Candidate, _ ParseOptions) (*Entry, error) {
	body, err := readCandidateFile(candidate)
	if err != nil {
		return nil, fmt.Errorf("read devhints file: %w", err)
	}

	topic := strings.TrimSpace(candidate.Topic)
	if topic == "" {
		topic = strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
	}

	return &Entry{
		ID:       entryID(SourceDevhints, topic, candidate.ID),
		Source:   SourceDevhints,
		Topic:    topic,
		Title:    topic,
		Tags:     []string{topic},
		Content:  body,
		Category: CategoryCheatsheet,
	}, nil
}
