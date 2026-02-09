package backend

import (
	"cheatr/internal/backend/parsers"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrResolutionNotFound = errors.New("resolution not found")

type routingResolver struct {
	sources   *SourceManager
	languages LanguageDetector
	loadCtx   context.Context
}

func newRoutingResolver(sources *SourceManager, detector LanguageDetector) Resolver {
	return &routingResolver{
		sources:   sources,
		languages: detector,
		loadCtx:   context.Background(),
	}
}

func (r *routingResolver) Resolve(args []string) (*Resolution, error) {
	if len(args) == 0 {
		return nil, errors.New("topic is required")
	}

	topic := normalizeTopic(args[0])
	if topic == "" {
		return nil, errors.New("topic is required")
	}

	if len(args) > 1 {
		return nil, notImplemented("ResolveSubtopic")
	}

	if r.IsLanguage(topic) {
		entry, err := r.lookupEntry(SourceLXIYM, topic)
		if err != nil {
			return nil, err
		}
		return toResolution(SourceLXIYM, topic, entry), nil
	}

	if r.HasTldrEntry(topic) {
		entry, err := r.lookupEntry(SourceTldr, topic)
		if err != nil {
			return nil, err
		}
		return toResolution(SourceTldr, topic, entry), nil
	}

	entry, err := r.lookupEntry(SourceDevhints, topic)
	if err != nil {
		return nil, err
	}

	return toResolution(SourceDevhints, topic, entry), nil
}

func (r *routingResolver) ResolveSubtopic(lang, subtopic string) (*Resolution, error) {
	return nil, notImplemented("ResolveSubtopic")
}

func (r *routingResolver) ResolveDocs(slug, search string) (*Resolution, error) {
	return nil, notImplemented("ResolveDocs")
}

func (r *routingResolver) Search(query string, filter SourceFilter) ([]SearchResult, error) {
	return nil, notImplemented("Search")
}

func (r *routingResolver) IsLanguage(name string) bool {
	if r.languages == nil {
		return false
	}
	return r.languages.IsLanguage(normalizeTopic(name))
}

func (r *routingResolver) HasTldrEntry(name string) bool {
	_, err := r.lookupEntry(SourceTldr, name)
	return err == nil
}

func (r *routingResolver) HasLocalDevDoc(slug string) bool {
	if r.sources == nil {
		return false
	}

	hasBundle, err := r.sources.HasDevDocBundle(slug)
	if err != nil {
		return false
	}

	return hasBundle
}

func (r *routingResolver) lookupEntry(source, topic string) (*parsers.Entry, error) {
	repoPath, err := r.sources.RepoPath(source)
	if err != nil {
		return nil, err
	}

	entries, err := loadEntriesBySource(r.loadCtx, source, repoPath)
	if err != nil {
		return nil, err
	}

	normalizedTopic := normalizeTopic(topic)
	var exactMatches []*parsers.Entry
	for _, entry := range entries {
		if normalizeTopic(entry.Topic) == normalizedTopic {
			exactMatches = append(exactMatches, entry)
		}
	}

	if len(exactMatches) == 0 {
		return nil, fmt.Errorf("%s %q: %w", source, topic, ErrResolutionNotFound)
	}

	if source == SourceTldr {
		for _, entry := range exactMatches {
			if strings.Contains(entry.ID, ":common:") {
				return entry, nil
			}
		}
	}

	return exactMatches[0], nil
}

func loadEntriesBySource(ctx context.Context, source, repoPath string) ([]*parsers.Entry, error) {
	switch source {
	case SourceLXIYM:
		return parsers.LoadLXIYMEntries(ctx, repoPath)
	case SourceDevhints:
		return parsers.LoadDevhintsEntries(ctx, repoPath)
	case SourceTldr:
		return parsers.LoadTLDREntries(ctx, repoPath)
	default:
		return nil, fmt.Errorf("unsupported source %q", source)
	}
}

func toResolution(source, topic string, entry *parsers.Entry) *Resolution {
	resolvedTopic := normalizeTopic(topic)
	if entry != nil {
		if t := normalizeTopic(entry.Topic); t != "" {
			resolvedTopic = t
		}
	}

	content := ""
	if entry != nil {
		content = entry.Content
	}

	return &Resolution{
		Source:  source,
		Topic:   resolvedTopic,
		Content: content,
	}
}

func normalizeTopic(topic string) string {
	return strings.ToLower(strings.TrimSpace(topic))
}
