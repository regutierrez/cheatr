package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
)

const devDocsCatalogURL = "https://devdocs.io/docs.json"

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
	return filepath.Join(m.dataDir, "devdocs", "docs.json")
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
