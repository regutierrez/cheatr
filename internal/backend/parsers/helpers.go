package parsers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

func discoverMarkdownFiles(ctx context.Context, source, root string, filter func(path string, d fs.DirEntry) bool, limit int) ([]Candidate, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("discover root is required")
	}

	candidates := make([]Candidate, 0, 256)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		if filter != nil && !filter(path, d) {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		topic := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		candidates = append(candidates, Candidate{
			Source: source,
			Kind:   CandidateFile,
			ID:     filepath.ToSlash(relPath),
			Path:   path,
			Topic:  topic,
			Title:  topic,
		})

		if limit > 0 && len(candidates) >= limit {
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, fs.SkipAll) {
			return candidates, nil
		}
		return nil, fmt.Errorf("discover markdown files: %w", err)
	}

	return candidates, nil
}

func loadWithParser(ctx context.Context, source string, candidates []Candidate, parseFn ParseFunc, opts LoadOptions) ([]*Entry, error) {
	entries := make([]*Entry, 0, len(candidates))
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		entry, err := parseFn(ctx, candidate, opts.Parse)
		if err != nil {
			if errors.Is(err, ErrSkipCandidate) {
				continue
			}
			if opts.ContinueOnError {
				continue
			}
			return nil, fmt.Errorf("parse %s candidate %q: %w", source, candidate.ID, err)
		}

		if entry == nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func readCandidateFile(candidate Candidate) (string, error) {
	if strings.TrimSpace(candidate.Path) == "" {
		return "", errors.New("candidate path is required")
	}

	body, err := os.ReadFile(candidate.Path)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func entryID(source, topic, fallback string) string {
	cleanTopic := strings.TrimSpace(topic)
	if cleanTopic != "" {
		return source + ":" + cleanTopic
	}
	return source + ":" + strings.TrimSpace(fallback)
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

func parseLevel3Sections(markdown string) []Section {
	if markdown == "" {
		return nil
	}

	sections := make([]Section, 0)
	current := -1
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		heading, ok := parseLevel3Heading(line)
		if ok {
			sections = append(sections, Section{Heading: heading})
			current = len(sections) - 1
			continue
		}

		if current == -1 {
			continue
		}

		if sections[current].Content == "" {
			sections[current].Content = line
			continue
		}
		sections[current].Content += "\n" + line
	}

	for i := range sections {
		sections[i].Content = strings.TrimSpace(sections[i].Content)
	}

	return sections
}

func parseLevel3Heading(line string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if !strings.HasPrefix(trimmed, "###") {
		return "", false
	}
	if len(trimmed) < 4 || trimmed[3] != ' ' {
		return "", false
	}

	heading := strings.TrimSpace(trimmed[4:])
	heading = strings.TrimSpace(strings.TrimRight(heading, "#"))
	if heading == "" {
		return "", false
	}

	return heading, true
}
