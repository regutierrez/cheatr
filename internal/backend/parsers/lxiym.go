package parsers

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type LXIYMLoader struct{}

func NewLXIYMLoader() *LXIYMLoader {
	return &LXIYMLoader{}
}

func (l *LXIYMLoader) Source() string {
	return SourceLXIYM
}

func (l *LXIYMLoader) Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error) {
	return discoverMarkdownFiles(ctx, l.Source(), opts.Root, func(path string, _ fs.DirEntry) bool {
		base := strings.ToLower(filepath.Base(path))
		return base != "readme.md"
	}, opts.Limit)
}

func (l *LXIYMLoader) Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error) {
	return ParseLXIYMCandidate(ctx, candidate, opts)
}

func (l *LXIYMLoader) Load(ctx context.Context, opts LoadOptions) ([]*Entry, error) {
	candidates, err := l.Discover(ctx, opts.Discover)
	if err != nil {
		return nil, err
	}

	return loadWithParser(ctx, l.Source(), candidates, l.Parse, opts)
}

func LoadLXIYMEntries(ctx context.Context, repoRoot string) ([]*Entry, error) {
	loader := NewLXIYMLoader()
	return loader.Load(ctx, LoadOptions{Discover: DiscoverOptions{Root: repoRoot}})
}

func ParseLXIYMCandidate(_ context.Context, candidate Candidate, _ ParseOptions) (*Entry, error) {
	raw, err := readCandidateFile(candidate)
	if err != nil {
		return nil, fmt.Errorf("read lxiym file: %w", err)
	}

	topic := strings.TrimSpace(candidate.Topic)
	if topic == "" {
		topic = strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
	}

	meta, body, err := parseYAMLFrontmatter(raw)
	if err != nil {
		return nil, fmt.Errorf("parse lxiym frontmatter: %w", err)
	}

	title := topic
	if v := firstMetaString(meta, "title", "name"); v != "" {
		title = v
	}

	tags := []string{topic}
	tags = append(tags, metaStringList(meta, "tags", "tag", "keywords")...)
	tags = append(tags, metaStringList(meta, "language", "lang")...)
	tags = dedupeStrings(tags)

	return &Entry{
		ID:       entryID(SourceLXIYM, topic, candidate.ID),
		Source:   SourceLXIYM,
		Topic:    topic,
		Title:    title,
		Tags:     tags,
		Content:  body,
		Category: CategorySyntax,
	}, nil
}
