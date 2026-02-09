package backend

import (
	"cheatr/internal/backend/parsers"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const parsedEntriesCacheVersion = 1

var parsedEntryCacheSources = map[string]struct{}{
	SourceLXIYM:    {},
	SourceDevhints: {},
	SourceTldr:     {},
}

type CacheManager struct {
	cacheRoot string
	mu        sync.RWMutex
}

type parsedEntriesCachePayload struct {
	Version  int              `json:"version"`
	Source   string           `json:"source"`
	RepoPath string           `json:"repo_path"`
	CachedAt time.Time        `json:"cached_at"`
	Entries  []*parsers.Entry `json:"entries"`
}

func NewCacheManager(dataDir string) *CacheManager {
	return &CacheManager{cacheRoot: filepath.Join(dataDir, "cache")}
}

func (m *CacheManager) LoadParsedEntries(source, repoPath string) ([]*parsers.Entry, bool, error) {
	if m == nil {
		return nil, false, nil
	}

	cachePath, ok := m.parsedEntriesCachePath(source)
	if !ok {
		return nil, false, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	body, err := os.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read parsed-entry cache: %w", err)
	}

	var payload parsedEntriesCachePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("decode parsed-entry cache: %w", err)
	}

	if payload.Version != parsedEntriesCacheVersion {
		return nil, false, nil
	}
	if payload.Source != normalizeTopic(source) {
		return nil, false, nil
	}
	if filepath.Clean(payload.RepoPath) != filepath.Clean(repoPath) {
		return nil, false, nil
	}

	entries := payload.Entries
	if entries == nil {
		entries = []*parsers.Entry{}
	}

	return entries, true, nil
}

func (m *CacheManager) SaveParsedEntries(source, repoPath string, entries []*parsers.Entry) error {
	if m == nil {
		return nil
	}

	cachePath, ok := m.parsedEntriesCachePath(source)
	if !ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("create parsed-entry cache dir: %w", err)
	}

	payload := parsedEntriesCachePayload{
		Version:  parsedEntriesCacheVersion,
		Source:   normalizeTopic(source),
		RepoPath: filepath.Clean(repoPath),
		CachedAt: time.Now().UTC(),
		Entries:  entries,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode parsed-entry cache: %w", err)
	}

	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		return fmt.Errorf("write parsed-entry cache temp file: %w", err)
	}

	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("finalize parsed-entry cache file: %w", err)
	}

	return nil
}

func (m *CacheManager) InvalidateParsedEntries(source string) error {
	if m == nil {
		return nil
	}

	cachePath, ok := m.parsedEntriesCachePath(source)
	if !ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.Remove(cachePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove parsed-entry cache: %w", err)
	}

	return nil
}

func (m *CacheManager) parsedEntriesCachePath(source string) (string, bool) {
	normalized := normalizeTopic(source)
	if _, ok := parsedEntryCacheSources[normalized]; !ok {
		return "", false
	}

	fileName := normalized + "_entries.json"
	return filepath.Join(m.cacheRoot, "parsed_entries", fileName), true
}

func cacheableParsedEntrySource(source string) bool {
	_, ok := parsedEntryCacheSources[strings.ToLower(strings.TrimSpace(source))]
	return ok
}
