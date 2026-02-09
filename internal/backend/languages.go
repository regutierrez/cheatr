package backend

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

func deriveLanguagesFromFilenames(repoRoot string) ([]string, error) {
	languages := make([]string, 0, 256)
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		base := strings.ToLower(filepath.Base(path))
		if filepath.Ext(base) != ".md" {
			return nil
		}
		if base == "readme.md" {
			return nil
		}

		language := normalizeLanguageName(strings.TrimSuffix(base, filepath.Ext(base)))
		if language != "" {
			languages = append(languages, language)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return dedupeAndSortLanguages(languages), nil
}

func normalizeLanguageName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func dedupeAndSortLanguages(languages []string) []string {
	seen := make(map[string]struct{}, len(languages))
	out := make([]string, 0, len(languages))
	for _, language := range languages {
		normalized := normalizeLanguageName(language)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	sort.Strings(out)
	return out
}
