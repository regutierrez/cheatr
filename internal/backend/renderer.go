package backend

import (
	"cheatr/internal/backend/parsers"
	"errors"
	"strings"
)

type contentRenderer interface {
	RenderEntry(entry *parsers.Entry) (string, error)
	RenderSection(entry *parsers.Entry, heading string) (string, error)
	RenderMarkdown(markdown string) (string, error)
}

type markdownRenderer struct{}

func newMarkdownRenderer() contentRenderer {
	return &markdownRenderer{}
}

func (r *markdownRenderer) RenderEntry(entry *parsers.Entry) (string, error) {
	if entry == nil {
		return "", errors.New("entry is required")
	}

	return r.RenderMarkdown(entry.Content)
}

func (r *markdownRenderer) RenderSection(entry *parsers.Entry, heading string) (string, error) {
	if entry == nil {
		return "", errors.New("entry is required")
	}

	sectionKey := normalizeLooseKey(heading)
	if sectionKey == "" {
		return "", errors.New("section heading is required")
	}

	for _, section := range entry.Sections {
		if normalizeLooseKey(section.Heading) != sectionKey {
			continue
		}

		content := "### " + strings.TrimSpace(section.Heading)
		if body := strings.TrimSpace(section.Content); body != "" {
			content += "\n\n" + body
		}

		return r.RenderMarkdown(content)
	}

	return "", ErrResolutionNotFound
}

func (r *markdownRenderer) RenderMarkdown(markdown string) (string, error) {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return "", ErrResolutionNotFound
	}

	return trimmed, nil
}
