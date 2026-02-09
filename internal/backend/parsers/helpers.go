package parsers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
