package parsers

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
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

func parseYAMLFrontmatter(raw string) (map[string]any, string, error) {
	content := strings.TrimPrefix(raw, "\ufeff")
	if !strings.HasPrefix(content, "---") {
		return nil, content, nil
	}

	firstLineEnd := strings.IndexByte(content, '\n')
	if firstLineEnd == -1 {
		if strings.TrimSpace(strings.TrimSuffix(content, "\r")) == "---" {
			return nil, "", nil
		}
		return nil, content, nil
	}

	if strings.TrimSpace(strings.TrimSuffix(content[:firstLineEnd], "\r")) != "---" {
		return nil, content, nil
	}

	fmStart := firstLineEnd + 1
	pos := fmStart
	for pos <= len(content) {
		lineStart := pos
		nextLineEnd := strings.IndexByte(content[pos:], '\n')
		if nextLineEnd == -1 {
			nextLineEnd = len(content) - pos
		}
		lineEnd := pos + nextLineEnd
		line := strings.TrimSpace(strings.TrimSuffix(content[lineStart:lineEnd], "\r"))
		if line == "---" {
			frontmatter := content[fmStart:lineStart]
			bodyStart := lineEnd
			if bodyStart < len(content) && content[bodyStart] == '\n' {
				bodyStart++
			}

			meta := map[string]any{}
			if strings.TrimSpace(frontmatter) != "" {
				if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
					return nil, "", err
				}
			}

			return meta, content[bodyStart:], nil
		}

		if lineEnd == len(content) {
			break
		}
		pos = lineEnd + 1
	}

	return nil, content, nil
}

func firstMetaString(meta map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := meta[key]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func metaStringList(meta map[string]any, keys ...string) []string {
	out := make([]string, 0)
	for _, key := range keys {
		value, ok := meta[key]
		if !ok {
			continue
		}

		switch v := value.(type) {
		case string:
			parts := strings.Split(v, ",")
			for _, part := range parts {
				if s := strings.TrimSpace(part); s != "" {
					out = append(out, s)
				}
			}
		case []any:
			for _, item := range v {
				if s := strings.TrimSpace(fmt.Sprint(item)); s != "" && s != "<nil>" {
					out = append(out, s)
				}
			}
		case []string:
			for _, item := range v {
				if s := strings.TrimSpace(item); s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if slices.ContainsFunc(out, func(existing string) bool {
			return strings.EqualFold(existing, trimmed)
		}) {
			continue
		}

		out = append(out, trimmed)
	}

	return out
}
