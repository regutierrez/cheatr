package backend

import (
	"cheatr/internal/backend/parsers"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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

	if topic == "docs" {
		if len(args) < 2 {
			return nil, errors.New("docs slug is required")
		}

		return r.ResolveDocs(args[1], strings.Join(args[2:], " "))
	}

	if len(args) > 1 {
		if r.IsLanguage(topic) {
			return r.ResolveSubtopic(topic, strings.Join(args[1:], " "))
		}
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
	lang = normalizeTopic(lang)
	subtopic = normalizeTopic(subtopic)
	if lang == "" {
		return nil, errors.New("language is required")
	}
	if subtopic == "" {
		return nil, errors.New("subtopic is required")
	}

	if res, ok, err := r.resolveDevhintsSubtopic(lang, subtopic); err != nil {
		return nil, err
	} else if ok {
		return res, nil
	}

	return r.resolveDevDocsSubtopic(lang, subtopic)
}

func (r *routingResolver) ResolveDocs(slug, search string) (*Resolution, error) {
	if r.sources == nil {
		return nil, errors.New("source manager is required")
	}

	normalizedSlug, err := normalizeDevDocSlug(slug)
	if err != nil {
		return nil, err
	}

	bundlePath, err := r.sources.EnsureDevDocBundle(normalizedSlug)
	if err != nil {
		return nil, err
	}

	indexEntries, err := loadDevDocsIndexEntries(bundlePath)
	if err != nil {
		return nil, err
	}

	trimmedSearch := strings.TrimSpace(search)
	if trimmedSearch == "" {
		return &Resolution{
			Source:     SourceDevDocs,
			Topic:      normalizedSlug,
			Candidates: collectDevDocsBrowseCandidates(indexEntries),
		}, nil
	}

	if exact := findDevDocsExactMatch(indexEntries, trimmedSearch); exact != nil {
		content, err := loadDevDocsPageContent(bundlePath, exact.path())
		if err != nil {
			return nil, err
		}

		return &Resolution{
			Source:   SourceDevDocs,
			Topic:    normalizedSlug,
			Subtopic: trimmedSearch,
			Content:  content,
		}, nil
	}

	candidates := collectDevDocsRelatedCandidates(indexEntries, trimmedSearch)
	if len(candidates) == 0 {
		return &Resolution{
			Source:   SourceDevDocs,
			Topic:    normalizedSlug,
			Subtopic: trimmedSearch,
			Content: fmt.Sprintf(
				"No results for '%s' in %s docs.\nTry: cheatr %s %s",
				trimmedSearch,
				normalizedSlug,
				normalizedSlug,
				trimmedSearch,
			),
		}, nil
	}

	return &Resolution{
		Source:     SourceDevDocs,
		Topic:      normalizedSlug,
		Subtopic:   trimmedSearch,
		Candidates: candidates,
	}, nil
}

func (r *routingResolver) Search(query string, filter SourceFilter) ([]SearchResult, error) {
	if r.sources == nil {
		return nil, errors.New("source manager is required")
	}

	searchQuery := strings.TrimSpace(query)
	sources, err := searchSourcesForFilter(filter)
	if err != nil {
		return nil, err
	}

	languageLike := r.queryLooksLanguageLike(searchQuery)
	cliLike := r.queryLooksCLILike(searchQuery)

	results := make([]SearchResult, 0, 256)
	for _, source := range sources {
		var sourceResults []SearchResult
		switch source {
		case SourceLXIYM, SourceDevhints, SourceTldr:
			sourceResults, err = r.searchRepoSource(source, searchQuery, languageLike, cliLike)
		case SourceDevDocs:
			sourceResults, err = r.searchLocalDevDocs(searchQuery, languageLike, cliLike)
		default:
			err = fmt.Errorf("unsupported search source %q", source)
		}
		if err != nil {
			return nil, err
		}
		results = append(results, sourceResults...)
	}

	if action, ok, err := r.localDevDocsBrowseAction(searchQuery, filter); err != nil {
		return nil, err
	} else if ok {
		results = append(results, action)
	}

	sort.SliceStable(results, func(i, j int) bool {
		leftScore := results[i].Score + sourceRoutingBoost(results[i].Priority)
		rightScore := results[j].Score + sourceRoutingBoost(results[j].Priority)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Priority != results[j].Priority {
			return results[i].Priority < results[j].Priority
		}
		leftTitle := ""
		rightTitle := ""
		if results[i].Entry != nil {
			leftTitle = strings.ToLower(strings.TrimSpace(results[i].Entry.Title))
		}
		if results[j].Entry != nil {
			rightTitle = strings.ToLower(strings.TrimSpace(results[j].Entry.Title))
		}
		if leftTitle != rightTitle {
			return leftTitle < rightTitle
		}
		return strings.ToLower(results[i].Source) < strings.ToLower(results[j].Source)
	})

	return results, nil
}

func (r *routingResolver) localDevDocsBrowseAction(query string, filter SourceFilter) (SearchResult, bool, error) {
	if filter != FilterNone && filter != FilterDevDocs {
		return SearchResult{}, false, nil
	}

	queryKey := normalizeDevDocsInjectKey(query)
	if queryKey == "" {
		return SearchResult{}, false, nil
	}

	localSlugs, err := r.localDevDocsBundleSlugs()
	if err != nil {
		return SearchResult{}, false, err
	}

	if len(localSlugs) == 0 {
		return SearchResult{}, false, nil
	}

	lookup := make(map[string]string, len(localSlugs))
	for _, slug := range localSlugs {
		lookup[slug] = slug
		lookup[normalizeDevDocsInjectKey(slug)] = slug
	}

	matchedSlug, ok := lookup[queryKey]
	if !ok {
		return SearchResult{}, false, nil
	}

	return SearchResult{
		Kind:   SearchAction,
		Label:  fmt.Sprintf("Browse %s docs (local)", matchedSlug),
		Source: SourceDevDocs,
		Action: ActionBrowseDevDocs,
		Meta:   map[string]string{"slug": matchedSlug},
	}, true, nil
}

func (r *routingResolver) localDevDocsBundleSlugs() ([]string, error) {
	bundlesDir := filepath.Join(r.sources.dataDir, "devdocs", "bundles")
	dirs, err := os.ReadDir(bundlesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list local devdocs bundles: %w", err)
	}

	slugs := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		slug, err := normalizeDevDocSlug(dir.Name())
		if err != nil {
			continue
		}

		slugs = append(slugs, slug)
	}

	return slugs, nil
}

func normalizeDevDocsInjectKey(value string) string {
	value = normalizeTopic(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == ' ', r == '-', r == '_', r == '.':
			continue
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}
func searchSourcesForFilter(filter SourceFilter) ([]string, error) {
	switch filter {
	case FilterNone:
		return []string{SourceLXIYM, SourceDevhints, SourceTldr, SourceDevDocs}, nil
	case FilterLXIYM:
		return []string{SourceLXIYM}, nil
	case FilterDevhints:
		return []string{SourceDevhints}, nil
	case FilterTldr:
		return []string{SourceTldr}, nil
	case FilterDevDocs:
		return []string{SourceDevDocs}, nil
	default:
		return nil, fmt.Errorf("unsupported source filter %q", filter)
	}
}

func (r *routingResolver) searchRepoSource(source, query string, languageLike, cliLike bool) ([]SearchResult, error) {
	repoPath, err := r.sources.RepoPath(source)
	if err != nil {
		return nil, err
	}

	entries, err := loadEntriesBySource(r.loadCtx, source, repoPath)
	if err != nil {
		return nil, err
	}

	priority := routingPriorityForSource(source, languageLike, cliLike)
	results := make([]SearchResult, 0, len(entries))
	for _, entry := range entries {
		score := scoreEntryMatch(query, entry, "", "")
		if strings.TrimSpace(query) != "" && score <= 0 {
			continue
		}

		results = append(results, SearchResult{
			Kind:     SearchEntry,
			Entry:    entry,
			Source:   source,
			Action:   ActionNone,
			Score:    score,
			Priority: priority,
		})
	}

	return results, nil
}

func (r *routingResolver) searchLocalDevDocs(query string, languageLike, cliLike bool) ([]SearchResult, error) {
	bundlesDir := filepath.Join(r.sources.dataDir, "devdocs", "bundles")
	dirs, err := os.ReadDir(bundlesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list local devdocs bundles: %w", err)
	}

	priority := routingPriorityForSource(SourceDevDocs, languageLike, cliLike)
	results := make([]SearchResult, 0, 128)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		slug, err := normalizeDevDocSlug(dir.Name())
		if err != nil {
			continue
		}

		bundlePath := filepath.Join(bundlesDir, slug)
		indexEntries, err := loadDevDocsIndexEntries(bundlePath)
		if err != nil {
			continue
		}

		for _, indexEntry := range indexEntries {
			entryPath := strings.TrimSpace(indexEntry.path())
			if entryPath == "" {
				continue
			}

			title := strings.TrimSpace(indexEntry.Name)
			if title == "" {
				title = strings.Trim(entryPath, "/")
			}

			entry := &parsers.Entry{
				ID:       fmt.Sprintf("%s:%s:%s", SourceDevDocs, slug, entryPath),
				Source:   SourceDevDocs,
				Topic:    slug,
				Title:    title,
				Category: parsers.CategoryAPI,
			}

			score := scoreEntryMatch(query, entry, slug, entryPath)
			if strings.TrimSpace(query) != "" && score <= 0 {
				continue
			}

			results = append(results, SearchResult{
				Kind:     SearchEntry,
				Entry:    entry,
				Source:   SourceDevDocs,
				Action:   ActionNone,
				Meta:     map[string]string{"slug": slug, "path": entryPath},
				Score:    score,
				Priority: priority,
			})
		}
	}

	return results, nil
}

func (r *routingResolver) queryLooksLanguageLike(query string) bool {
	normalized := normalizeTopic(query)
	if normalized == "" {
		return false
	}

	if strings.Contains(normalized, " ") {
		parts := strings.Fields(normalized)
		if len(parts) == 0 {
			return false
		}
		normalized = parts[0]
	}

	return r.IsLanguage(normalized)
}

func (r *routingResolver) queryLooksCLILike(query string) bool {
	normalized := normalizeTopic(query)
	if normalized == "" {
		return false
	}

	if strings.ContainsAny(normalized, "|><") {
		return true
	}

	if strings.Contains(normalized, "--") || strings.Contains(normalized, " -") {
		return true
	}

	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return false
	}

	if r.HasTldrEntry(parts[0]) {
		return true
	}

	if len(parts) > 1 && !r.IsLanguage(parts[0]) {
		return true
	}

	return false
}

func routingPriorityForSource(source string, languageLike, cliLike bool) int {
	switch {
	case languageLike:
		switch source {
		case SourceLXIYM:
			return 0
		case SourceDevhints:
			return 1
		case SourceDevDocs:
			return 2
		case SourceTldr:
			return 3
		}
	case cliLike:
		switch source {
		case SourceTldr:
			return 0
		case SourceDevhints:
			return 1
		case SourceLXIYM:
			return 2
		case SourceDevDocs:
			return 3
		}
	default:
		switch source {
		case SourceDevhints:
			return 0
		case SourceTldr:
			return 1
		case SourceLXIYM:
			return 2
		case SourceDevDocs:
			return 3
		}
	}

	return 4
}

func sourceRoutingBoost(priority int) int {
	switch priority {
	case 0:
		return 36
	case 1:
		return 24
	case 2:
		return 12
	default:
		return 0
	}
}

func scoreEntryMatch(query string, entry *parsers.Entry, extraTerms ...string) int {
	needle := normalizeLooseKey(query)
	if needle == "" {
		return 1
	}

	terms := make([]string, 0, len(entry.Tags)+6+len(extraTerms))
	terms = append(terms, entry.Title, entry.Topic, entry.ID, entry.Category)
	terms = append(terms, entry.Tags...)
	terms = append(terms, extraTerms...)

	best := 0
	for _, term := range terms {
		candidate := normalizeLooseKey(term)
		if candidate == "" {
			continue
		}
		score := scoreLooseMatch(needle, candidate)
		if score > best {
			best = score
		}
	}

	return best
}

func scoreLooseMatch(needle, candidate string) int {
	if needle == "" || candidate == "" {
		return 0
	}

	if needle == candidate {
		return 220
	}

	if strings.HasPrefix(candidate, needle) {
		return 180 - minInt(30, len(candidate)-len(needle))
	}

	if idx := strings.Index(candidate, needle); idx >= 0 {
		return 140 - minInt(40, idx)
	}

	if !isSubsequence(needle, candidate) {
		return 0
	}

	return 70 - minInt(30, len(candidate)-len(needle))
}

func isSubsequence(needle, candidate string) bool {
	if len(needle) == 0 {
		return true
	}

	idx := 0
	for i := 0; i < len(candidate) && idx < len(needle); i++ {
		if candidate[i] == needle[idx] {
			idx++
		}
	}

	return idx == len(needle)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func (r *routingResolver) resolveDevhintsSubtopic(lang, subtopic string) (*Resolution, bool, error) {
	repoPath, err := r.sources.RepoPath(SourceDevhints)
	if err != nil {
		return nil, false, err
	}

	entries, err := loadEntriesBySource(r.loadCtx, SourceDevhints, repoPath)
	if err != nil {
		return nil, false, err
	}

	exactTopic := normalizeDashedKey(lang + "-" + subtopic)
	languageTopic := normalizeTopic(lang)
	sectionKey := normalizeLooseKey(subtopic)

	var languageEntry *parsers.Entry
	for _, entry := range entries {
		if normalizeDashedKey(entry.Topic) == exactTopic {
			return &Resolution{
				Source:   SourceDevhints,
				Topic:    languageTopic,
				Subtopic: subtopic,
				Content:  entry.Content,
			}, true, nil
		}

		if languageEntry == nil && normalizeTopic(entry.Topic) == languageTopic {
			languageEntry = entry
		}
	}

	if languageEntry == nil {
		return nil, false, nil
	}

	for _, section := range languageEntry.Sections {
		if normalizeLooseKey(section.Heading) != sectionKey {
			continue
		}

		content := "### " + strings.TrimSpace(section.Heading)
		if trimmed := strings.TrimSpace(section.Content); trimmed != "" {
			content += "\n\n" + trimmed
		}

		return &Resolution{
			Source:   SourceDevhints,
			Topic:    languageTopic,
			Subtopic: subtopic,
			Content:  content,
		}, true, nil
	}

	return nil, false, nil
}

func (r *routingResolver) resolveDevDocsSubtopic(lang, subtopic string) (*Resolution, error) {
	slug, err := r.pickDevDocsSlugForLanguage(lang)
	if err != nil {
		return nil, err
	}

	bundlePath, err := r.sources.EnsureDevDocBundle(slug)
	if err != nil {
		return nil, err
	}

	indexEntries, err := loadDevDocsIndexEntries(bundlePath)
	if err != nil {
		return nil, err
	}

	if exact := findDevDocsExactMatch(indexEntries, subtopic); exact != nil {
		content, err := loadDevDocsPageContent(bundlePath, exact.path())
		if err == nil {
			return &Resolution{
				Source:   SourceDevDocs,
				Topic:    normalizeTopic(lang),
				Subtopic: subtopic,
				Content:  content,
			}, nil
		}
	}

	return &Resolution{
		Source:     SourceDevDocs,
		Topic:      normalizeTopic(lang),
		Subtopic:   subtopic,
		Candidates: collectDevDocsRelatedCandidates(indexEntries, subtopic),
	}, nil
}

func (r *routingResolver) pickDevDocsSlugForLanguage(lang string) (string, error) {
	if r.sources == nil {
		return "", errors.New("source manager is required")
	}

	docs, err := r.sources.ListDevDocs()
	if err != nil {
		return "", err
	}

	langTopic := normalizeTopic(lang)
	langDashed := normalizeDashedKey(lang)
	langLoose := normalizeLooseKey(lang)

	bestSlug := ""
	bestScore := 0
	for _, doc := range docs {
		slug := normalizeTopic(doc.Slug)
		name := normalizeTopic(doc.Name)
		if slug == "" {
			continue
		}

		slugDashed := normalizeDashedKey(slug)
		nameDashed := normalizeDashedKey(name)
		slugLoose := normalizeLooseKey(slug)
		nameLoose := normalizeLooseKey(name)

		score := 0
		switch {
		case slug == langTopic:
			score = 100
		case slugDashed == langDashed:
			score = 95
		case nameDashed == langDashed:
			score = 90
		case nameLoose == langLoose:
			score = 85
		case slugLoose == langLoose:
			score = 80
		case strings.Contains(slugDashed, langDashed) || strings.Contains(nameDashed, langDashed):
			score = 60
		case strings.Contains(slugLoose, langLoose) || strings.Contains(nameLoose, langLoose):
			score = 50
		}

		if score > bestScore {
			bestScore = score
			bestSlug = slug
		}
	}

	if bestSlug == "" {
		return "", fmt.Errorf("no devdocs docset for language %q: %w", lang, ErrResolutionNotFound)
	}

	return bestSlug, nil
}

type devDocsIndexEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Slug string `json:"slug"`
}

func (e devDocsIndexEntry) path() string {
	if strings.TrimSpace(e.Path) != "" {
		return e.Path
	}
	return e.Slug
}

func loadDevDocsIndexEntries(bundlePath string) ([]devDocsIndexEntry, error) {
	body, err := os.ReadFile(filepath.Join(bundlePath, devDocsIndexFile))
	if err != nil {
		return nil, fmt.Errorf("read devdocs index: %w", err)
	}

	entries := make([]devDocsIndexEntry, 0, 1024)
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode devdocs index: %w", err)
	}

	if err := parseDevDocsIndex(payload, &entries); err != nil {
		return nil, fmt.Errorf("decode devdocs index: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("devdocs index has no entries: %w", ErrResolutionNotFound)
	}

	return dedupeDevDocsIndexEntries(entries), nil
}

func parseDevDocsIndex(node any, out *[]devDocsIndexEntry) error {
	switch typed := node.(type) {
	case []any:
		for _, item := range typed {
			if err := parseDevDocsIndex(item, out); err != nil {
				return err
			}
		}
	case map[string]any:
		name, _ := typed["name"].(string)
		entryPath, _ := typed["path"].(string)
		slug, _ := typed["slug"].(string)
		if strings.TrimSpace(name) != "" && (strings.TrimSpace(entryPath) != "" || strings.TrimSpace(slug) != "") {
			*out = append(*out, devDocsIndexEntry{Name: name, Path: entryPath, Slug: slug})
		}

		for _, value := range typed {
			if err := parseDevDocsIndex(value, out); err != nil {
				return err
			}
		}
	case nil, string, float64, bool:
		return nil
	default:
		return fmt.Errorf("unsupported devdocs index payload type %T", typed)
	}

	return nil
}

func dedupeDevDocsIndexEntries(entries []devDocsIndexEntry) []devDocsIndexEntry {
	type scoredEntry struct {
		entry devDocsIndexEntry
		score int
	}

	byPath := make(map[string]scoredEntry, len(entries))
	for _, entry := range entries {
		entryPath := normalizeTopic(entry.path())
		if entryPath == "" {
			continue
		}

		score := len(strings.TrimSpace(entry.Name))
		if existing, ok := byPath[entryPath]; !ok || score > existing.score {
			byPath[entryPath] = scoredEntry{entry: entry, score: score}
		}
	}

	out := make([]devDocsIndexEntry, 0, len(byPath))
	for _, pair := range byPath {
		out = append(out, pair.entry)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].path() < out[j].path()
	})

	return out
}

func findDevDocsExactMatch(entries []devDocsIndexEntry, subtopic string) *devDocsIndexEntry {
	needle := normalizeLooseKey(subtopic)
	if needle == "" {
		return nil
	}

	for i := range entries {
		entry := &entries[i]
		entryPath := normalizeTopic(entry.path())
		base := path.Base(strings.Trim(entryPath, "/"))
		if normalizeLooseKey(entry.Name) == needle || normalizeLooseKey(entryPath) == needle || normalizeLooseKey(base) == needle {
			return entry
		}
	}

	return nil
}

func collectDevDocsRelatedCandidates(entries []devDocsIndexEntry, subtopic string) []Candidate {
	type scoredCandidate struct {
		candidate Candidate
		score     int
	}

	needle := normalizeLooseKey(subtopic)
	if needle == "" {
		return nil
	}

	scored := make([]scoredCandidate, 0, 64)
	for _, entry := range entries {
		entryPath := strings.TrimSpace(entry.path())
		if entryPath == "" {
			continue
		}

		nameKey := normalizeLooseKey(entry.Name)
		pathKey := normalizeLooseKey(entryPath)
		if !strings.Contains(nameKey, needle) && !strings.Contains(pathKey, needle) {
			continue
		}

		score := 1
		if strings.HasPrefix(nameKey, needle) {
			score += 3
		}
		if strings.HasPrefix(pathKey, needle) {
			score += 2
		}

		title := strings.TrimSpace(entry.Name)
		if title == "" {
			title = strings.Trim(strings.TrimSpace(entryPath), "/")
		}

		scored = append(scored, scoredCandidate{
			candidate: Candidate{Title: title, Path: entryPath},
			score:     score,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].candidate.Title < scored[j].candidate.Title
	})

	if len(scored) > 25 {
		scored = scored[:25]
	}

	result := make([]Candidate, 0, len(scored))
	for _, item := range scored {
		result = append(result, item.candidate)
	}

	return result
}

func collectDevDocsBrowseCandidates(entries []devDocsIndexEntry) []Candidate {
	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		entryPath := strings.TrimSpace(entry.path())
		if entryPath == "" {
			continue
		}

		title := strings.TrimSpace(entry.Name)
		if title == "" {
			title = strings.Trim(entryPath, "/")
		}

		candidates = append(candidates, Candidate{Title: title, Path: entryPath})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Title != candidates[j].Title {
			return candidates[i].Title < candidates[j].Title
		}
		return candidates[i].Path < candidates[j].Path
	})

	return candidates
}

func loadDevDocsPageContent(bundlePath, entryPath string) (string, error) {
	records, err := loadDevDocsDBRecords(bundlePath)
	if err != nil {
		return "", err
	}

	if len(records) == 0 {
		return "", fmt.Errorf("devdocs bundle is empty: %w", ErrResolutionNotFound)
	}

	for _, key := range devDocsDBLookupKeys(entryPath) {
		html, ok := records[key]
		if !ok {
			continue
		}
		if strings.TrimSpace(html) == "" {
			continue
		}

		markdown, err := parsers.ConvertDevDocsHTMLToMarkdown(html)
		if err != nil {
			return "", err
		}

		return markdown, nil
	}

	return "", fmt.Errorf("devdocs page %q: %w", entryPath, ErrResolutionNotFound)
}

func loadDevDocsDBRecords(bundlePath string) (map[string]string, error) {
	body, err := os.ReadFile(filepath.Join(bundlePath, devDocsDBFile))
	if err != nil {
		return nil, fmt.Errorf("read devdocs db: %w", err)
	}

	var records map[string]string
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("decode devdocs db: %w", err)
	}

	return records, nil
}

func devDocsDBLookupKeys(entryPath string) []string {
	trimmed := strings.TrimSpace(entryPath)
	if trimmed == "" {
		return nil
	}

	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil
	}

	keys := []string{
		trimmed,
		"/" + trimmed,
	}

	withAnchor := strings.ReplaceAll(trimmed, "#", "/")
	keys = append(keys, withAnchor, "/"+withAnchor)

	return dedupeStringValues(keys)
}

func dedupeStringValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeDashedKey(value string) string {
	value = normalizeTopic(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}

		if lastDash {
			continue
		}
		b.WriteByte('-')
		lastDash = true
	}

	return strings.Trim(b.String(), "-")
}

func normalizeLooseKey(value string) string {
	value = normalizeTopic(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}
