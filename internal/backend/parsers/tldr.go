package parsers

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type TldrLoader struct{}

func NewTldrLoader() *TldrLoader {
	return &TldrLoader{}
}

func (l *TldrLoader) Source() string {
	return SourceTldr
}

func (l *TldrLoader) Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error) {
	root := opts.Root
	pagesRoot := filepath.Join(root, "pages")
	if info, err := os.Stat(pagesRoot); err == nil && info.IsDir() {
		root = pagesRoot
	}

	return discoverMarkdownFiles(ctx, l.Source(), root, func(path string, _ fs.DirEntry) bool {
		return strings.ToLower(filepath.Base(path)) != "readme.md"
	}, opts.Limit)
}

func (l *TldrLoader) Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error) {
	return ParseTldrCandidate(ctx, candidate, opts)
}

func (l *TldrLoader) Load(ctx context.Context, opts LoadOptions) ([]*Entry, error) {
	candidates, err := l.Discover(ctx, opts.Discover)
	if err != nil {
		return nil, err
	}

	return loadWithParser(ctx, l.Source(), candidates, l.Parse, opts)
}

func LoadTLDREntries(ctx context.Context, repoRoot string) ([]*Entry, error) {
	loader := NewTldrLoader()
	return loader.Load(ctx, LoadOptions{Discover: DiscoverOptions{Root: repoRoot}})
}

func ParseTldrCandidate(_ context.Context, candidate Candidate, _ ParseOptions) (*Entry, error) {
	body, err := readCandidateFile(candidate)
	if err != nil {
		return nil, fmt.Errorf("read tldr file: %w", err)
	}

	topic := strings.TrimSpace(candidate.Topic)
	if topic == "" {
		topic = strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
	}

	title := topic
	if parts := strings.Split(candidate.ID, "/"); len(parts) >= 2 {
		title = topic + " (" + parts[len(parts)-2] + ")"
	}

	return &Entry{
		ID:       entryID(SourceTldr, "", strings.ReplaceAll(candidate.ID, "/", ":")),
		Source:   SourceTldr,
		Topic:    topic,
		Title:    title,
		Tags:     []string{topic},
		Content:  body,
		Category: CategoryCommand,
	}, nil
}
