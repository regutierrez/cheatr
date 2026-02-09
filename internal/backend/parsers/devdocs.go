package parsers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	devDocsDefaultDBFile = "db.json"
)

type DevDocsLoader struct{}

func NewDevDocsLoader() *DevDocsLoader {
	return &DevDocsLoader{}
}

func (l *DevDocsLoader) Source() string {
	return SourceDevDocs
}

func (l *DevDocsLoader) Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error) {
	records, err := readDevDocsDB(opts.Root)
	if err != nil {
		return nil, err
	}

	candidates := make([]Candidate, 0, len(records))
	for id, payload := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		topic := normalizeDevDocsTopic(id)
		candidates = append(candidates, Candidate{
			Source:  SourceDevDocs,
			Kind:    CandidateRecord,
			ID:      id,
			Path:    opts.Root,
			Topic:   topic,
			Title:   topic,
			Payload: payload,
		})

		if opts.Limit > 0 && len(candidates) >= opts.Limit {
			break
		}
	}

	return candidates, nil
}

func (l *DevDocsLoader) Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error) {
	return ParseDevDocsCandidate(ctx, candidate, opts)
}

func (l *DevDocsLoader) Load(ctx context.Context, opts LoadOptions) ([]*Entry, error) {
	candidates, err := l.Discover(ctx, opts.Discover)
	if err != nil {
		return nil, err
	}

	return loadWithParser(ctx, l.Source(), candidates, l.Parse, opts)
}

func LoadDevDocsEntries(ctx context.Context, bundlePath string) ([]*Entry, error) {
	loader := NewDevDocsLoader()
	return loader.Load(ctx, LoadOptions{Discover: DiscoverOptions{Root: bundlePath}})
}

func ParseDevDocsCandidate(_ context.Context, candidate Candidate, _ ParseOptions) (*Entry, error) {
	payload := strings.TrimSpace(candidate.Payload)
	if payload == "" {
		return nil, ErrSkipCandidate
	}

	topic := strings.TrimSpace(candidate.Topic)
	if topic == "" {
		topic = normalizeDevDocsTopic(candidate.ID)
	}

	return &Entry{
		ID:       entryID(SourceDevDocs, topic, candidate.ID),
		Source:   SourceDevDocs,
		Topic:    topic,
		Title:    topic,
		Tags:     []string{topic},
		Content:  payload,
		Category: CategoryAPI,
	}, nil
}

func readDevDocsDB(root string) (map[string]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("devdocs root is required")
	}

	dbPath := root
	if filepath.Ext(dbPath) != ".json" {
		dbPath = filepath.Join(root, devDocsDefaultDBFile)
	}

	body, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("read devdocs db %q: %w", dbPath, err)
	}

	var records map[string]string
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("decode devdocs db %q: %w", dbPath, err)
	}

	return records, nil
}

func normalizeDevDocsTopic(id string) string {
	topic := strings.TrimPrefix(strings.TrimSpace(id), "/")
	topic = strings.TrimSuffix(topic, "/")
	if topic == "" {
		return "index"
	}
	return topic
}
