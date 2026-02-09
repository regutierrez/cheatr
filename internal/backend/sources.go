package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
)

const devDocsCatalogURL = "https://devdocs.io/docs.json"

const devDocsDocumentsBaseURL = "https://documents.devdocs.io"

const (
	devDocsCatalogFile = "docs.json"
	devDocsIndexFile   = "index.json"
	devDocsDBFile      = "db.json"
	devDocsStateFile   = "enabled.json"
)

type gitSource struct {
	Name string
	URL  string
	Dir  string
}

var defaultGitSources = []gitSource{
	{Name: SourceLXIYM, URL: "https://github.com/adambard/learnxinyminutes-docs.git", Dir: "lxiym"},
	{Name: SourceDevhints, URL: "https://github.com/rstacruz/cheatsheets.git", Dir: "devhints"},
	{Name: SourceTldr, URL: "https://github.com/tldr-pages/tldr.git", Dir: "tldr"},
}

type DevDoc struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type devDocsState struct {
	Enabled []string `json:"enabled"`
}

type SourceManager struct {
	dataDir    string
	httpClient *http.Client
	gitSources map[string]gitSource
}

func NewSourceManager(dataDir string) (*SourceManager, error) {
	resolved, err := resolveDataDir(dataDir)
	if err != nil {
		return nil, err
	}

	gitSources := make(map[string]gitSource, len(defaultGitSources))
	for _, src := range defaultGitSources {
		gitSources[src.Name] = src
	}

	return &SourceManager{
		dataDir: resolved,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		gitSources: gitSources,
	}, nil
}

func (m *SourceManager) Init() error {
	return m.updateAll()
}

func (m *SourceManager) Update() error {
	return m.updateAll()
}

func (m *SourceManager) UpdateSource(name string) error {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return errors.New("source name is required")
	}

	if normalized == SourceDevDocs {
		return m.fetchDevDocsCatalog()
	}

	src, ok := m.gitSources[normalized]
	if !ok {
		return fmt.Errorf("unknown source %q", name)
	}

	return m.cloneOrPull(src)
}

func (m *SourceManager) ListDevDocs() ([]DevDoc, error) {
	data, err := os.ReadFile(m.devDocsCatalogPath())
	if err != nil {
		return nil, fmt.Errorf("read devdocs catalog: %w", err)
	}

	var docs []DevDoc
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("decode devdocs catalog: %w", err)
	}

	return docs, nil
}

func (m *SourceManager) RepoPath(name string) (string, error) {
	src, ok := m.gitSources[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", fmt.Errorf("unknown source %q", name)
	}

	return filepath.Join(m.dataDir, src.Dir), nil
}

func (m *SourceManager) DevDocsCatalogPath() string {
	return m.devDocsCatalogPath()
}

func (m *SourceManager) EnableDevDoc(slug string) error {
	normalized, err := normalizeDevDocSlug(slug)
	if err != nil {
		return err
	}

	if err := m.ensureDevDocKnown(normalized); err != nil {
		return err
	}

	state, err := m.readDevDocsState()
	if err != nil {
		return err
	}

	if slices.Contains(state.Enabled, normalized) {
		return nil
	}

	state.Enabled = append(state.Enabled, normalized)
	return m.writeDevDocsState(state)
}

func (m *SourceManager) DisableDevDoc(slug string) error {
	normalized, err := normalizeDevDocSlug(slug)
	if err != nil {
		return err
	}

	state, err := m.readDevDocsState()
	if err != nil {
		return err
	}

	next := make([]string, 0, len(state.Enabled))
	for _, enabled := range state.Enabled {
		if enabled != normalized {
			next = append(next, enabled)
		}
	}
	state.Enabled = next

	return m.writeDevDocsState(state)
}

func (m *SourceManager) ListEnabledDevDocs() ([]string, error) {
	state, err := m.readDevDocsState()
	if err != nil {
		return nil, err
	}

	enabled := append([]string(nil), state.Enabled...)
	sort.Strings(enabled)
	return enabled, nil
}

func (m *SourceManager) EnsureDevDocBundle(slug string) (string, error) {
	normalized, err := normalizeDevDocSlug(slug)
	if err != nil {
		return "", err
	}

	if err := m.ensureDevDocKnown(normalized); err != nil {
		return "", err
	}

	bundlePath := m.devDocsBundlePath(normalized)
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		return "", fmt.Errorf("create devdocs bundle dir: %w", err)
	}

	indexPath := filepath.Join(bundlePath, devDocsIndexFile)
	if _, err := os.Stat(indexPath); errors.Is(err, os.ErrNotExist) {
		if err := m.downloadDevDocBundleFile(normalized, devDocsIndexFile, indexPath); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", fmt.Errorf("stat devdocs index bundle: %w", err)
	}

	dbPath := filepath.Join(bundlePath, devDocsDBFile)
	if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		if err := m.downloadDevDocBundleFile(normalized, devDocsDBFile, dbPath); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", fmt.Errorf("stat devdocs db bundle: %w", err)
	}

	return bundlePath, nil
}

func (m *SourceManager) DevDocBundlePath(slug string) (string, error) {
	normalized, err := normalizeDevDocSlug(slug)
	if err != nil {
		return "", err
	}

	return m.devDocsBundlePath(normalized), nil
}

func (m *SourceManager) HasDevDocBundle(slug string) (bool, error) {
	normalized, err := normalizeDevDocSlug(slug)
	if err != nil {
		return false, err
	}

	bundlePath := m.devDocsBundlePath(normalized)
	indexPath := filepath.Join(bundlePath, devDocsIndexFile)
	if _, err := os.Stat(indexPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat devdocs index bundle: %w", err)
	}

	dbPath := filepath.Join(bundlePath, devDocsDBFile)
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat devdocs db bundle: %w", err)
	}

	return true, nil
}

func (m *SourceManager) updateAll() error {
	for _, src := range defaultGitSources {
		if err := m.cloneOrPull(src); err != nil {
			return err
		}
	}

	return m.fetchDevDocsCatalog()
}

func (m *SourceManager) cloneOrPull(src gitSource) error {
	repoPath := filepath.Join(m.dataDir, src.Dir)
	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir %q: %w", m.dataDir, err)
	}

	if _, err := os.Stat(repoPath); errors.Is(err, os.ErrNotExist) {
		if _, err := git.PlainClone(repoPath, false, &git.CloneOptions{URL: src.URL, Depth: 1}); err != nil {
			return fmt.Errorf("clone %s: %w", src.Name, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("stat %q: %w", repoPath, err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("open %s repo: %w", src.Name, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("open %s worktree: %w", src.Name, err)
	}

	if err := wt.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull %s: %w", src.Name, err)
	}

	return nil
}

func (m *SourceManager) fetchDevDocsCatalog() error {
	if err := os.MkdirAll(filepath.Dir(m.devDocsCatalogPath()), 0o755); err != nil {
		return fmt.Errorf("create devdocs dir: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, devDocsCatalogURL, nil)
	if err != nil {
		return fmt.Errorf("create devdocs request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch devdocs catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch devdocs catalog: status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read devdocs catalog response: %w", err)
	}

	var docs []DevDoc
	if err := json.Unmarshal(body, &docs); err != nil {
		return fmt.Errorf("decode devdocs catalog response: %w", err)
	}

	if err := os.WriteFile(m.devDocsCatalogPath(), body, 0o644); err != nil {
		return fmt.Errorf("write devdocs catalog: %w", err)
	}

	return nil
}

func (m *SourceManager) devDocsCatalogPath() string {
	return filepath.Join(m.dataDir, "devdocs", devDocsCatalogFile)
}

func (m *SourceManager) devDocsStatePath() string {
	return filepath.Join(m.dataDir, "devdocs", devDocsStateFile)
}

func (m *SourceManager) devDocsBundlePath(slug string) string {
	return filepath.Join(m.dataDir, "devdocs", "bundles", slug)
}

func (m *SourceManager) ensureDevDocKnown(slug string) error {
	docs, err := m.ListDevDocs()
	if err != nil {
		return fmt.Errorf("load devdocs catalog: %w", err)
	}

	for _, doc := range docs {
		if strings.EqualFold(strings.TrimSpace(doc.Slug), slug) {
			return nil
		}
	}

	return fmt.Errorf("unknown devdocs slug %q", slug)
}

func (m *SourceManager) readDevDocsState() (devDocsState, error) {
	path := m.devDocsStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return devDocsState{}, nil
		}
		return devDocsState{}, fmt.Errorf("read devdocs state: %w", err)
	}

	var state devDocsState
	if err := json.Unmarshal(data, &state); err != nil {
		return devDocsState{}, fmt.Errorf("decode devdocs state: %w", err)
	}

	normalized := make([]string, 0, len(state.Enabled))
	for _, slug := range state.Enabled {
		clean, err := normalizeDevDocSlug(slug)
		if err != nil {
			continue
		}
		if slices.Contains(normalized, clean) {
			continue
		}
		normalized = append(normalized, clean)
	}
	sort.Strings(normalized)
	state.Enabled = normalized

	return state, nil
}

func (m *SourceManager) writeDevDocsState(state devDocsState) error {
	if err := os.MkdirAll(filepath.Dir(m.devDocsStatePath()), 0o755); err != nil {
		return fmt.Errorf("create devdocs state dir: %w", err)
	}

	state.Enabled = append([]string(nil), state.Enabled...)
	sort.Strings(state.Enabled)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode devdocs state: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(m.devDocsStatePath(), data, 0o644); err != nil {
		return fmt.Errorf("write devdocs state: %w", err)
	}

	return nil
}

func (m *SourceManager) downloadDevDocBundleFile(slug, fileName, destination string) error {
	fileURL, err := url.JoinPath(devDocsDocumentsBaseURL, slug, fileName)
	if err != nil {
		return fmt.Errorf("build devdocs bundle url: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("create devdocs bundle request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch devdocs bundle %q: %w", fileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch devdocs bundle %q: status %s", fileName, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read devdocs bundle %q response: %w", fileName, err)
	}

	if !json.Valid(body) {
		return fmt.Errorf("fetch devdocs bundle %q: invalid json", fileName)
	}

	tmpPath := destination + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		return fmt.Errorf("write temp devdocs bundle %q: %w", fileName, err)
	}

	if err := os.Rename(tmpPath, destination); err != nil {
		return fmt.Errorf("finalize devdocs bundle %q: %w", fileName, err)
	}

	return nil
}

func normalizeDevDocSlug(slug string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(slug))
	if normalized == "" {
		return "", errors.New("devdocs slug is required")
	}

	if strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") || strings.Contains(normalized, "..") {
		return "", fmt.Errorf("invalid devdocs slug %q", slug)
	}

	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', '~':
		default:
			return "", fmt.Errorf("invalid devdocs slug %q", slug)
		}
	}

	return normalized, nil
}

func resolveDataDir(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) != "" {
		return dataDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(homeDir, ".local", "share", "cheatr"), nil
}
