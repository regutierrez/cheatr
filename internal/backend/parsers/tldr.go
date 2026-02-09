package parsers

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type TldrLoader struct{}

func NewTldrLoader() *TldrLoader {
	return &TldrLoader{}
}

func (l *TldrLoader) Source() string {
	return SourceTldr
}

func (l *TldrLoader) Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error) {
	root := opts.Root
	pagesRoot := filepath.Join(root, "pages")
	if info, err := os.Stat(pagesRoot); err == nil && info.IsDir() {
		root = pagesRoot
	}

	return discoverMarkdownFiles(ctx, l.Source(), root, func(path string, _ fs.DirEntry) bool {
		return strings.ToLower(filepath.Base(path)) != "readme.md"
	}, opts.Limit)
}

func (l *TldrLoader) Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error) {
	return ParseTldrCandidate(ctx, candidate, opts)
}

func (l *TldrLoader) Load(ctx context.Context, opts LoadOptions) ([]*Entry, error) {
	candidates, err := l.Discover(ctx, opts.Discover)
	if err != nil {
		return nil, err
	}

	return loadWithParser(ctx, l.Source(), candidates, l.Parse, opts)
}

func LoadTLDREntries(ctx context.Context, repoRoot string) ([]*Entry, error) {
	loader := NewTldrLoader()
	return loader.Load(ctx, LoadOptions{Discover: DiscoverOptions{Root: repoRoot}})
}

func ParseTldrCandidate(_ context.Context, candidate Candidate, _ ParseOptions) (*Entry, error) {
	raw, err := readCandidateFile(candidate)
	if err != nil {
		return nil, fmt.Errorf("read tldr file: %w", err)
	}

	parsed := parseTLDRTemplate(raw)

	command := strings.TrimSpace(candidate.Topic)
	if command == "" {
		command = strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
	}
	if command == "" {
		command = deriveTLDRCommand(parsed.Title)
	}

	topic := command
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		title = topic
	}
	if parsed.Title == "" {
		parsed.Title = title
		parsed.Markdown = renderTLDRMarkdown(parsed)
	}

	platform := tldrPlatformTag(candidate.ID)
	tags := []string{topic}
	if parsedTitleCommand := deriveTLDRCommand(title); parsedTitleCommand != "" {
		tags = append(tags, parsedTitleCommand)
	}
	if title != "" {
		tags = append(tags, title)
	}
	if platform != "" {
		tags = append(tags, platform)
	}
	tags = dedupeStrings(tags)

	return &Entry{
		ID:       entryID(SourceTldr, "", strings.ReplaceAll(candidate.ID, "/", ":")),
		Source:   SourceTldr,
		Topic:    topic,
		Title:    title,
		Tags:     tags,
		Content:  parsed.Markdown,
		Category: CategoryCommand,
	}, nil
}

type tldrTemplate struct {
	Title       string
	Description []string
	Examples    []tldrExample
	Markdown    string
}

type tldrExample struct {
	Label string
	Code  []string
}

func parseTLDRTemplate(raw string) tldrTemplate {
	parsed := tldrTemplate{}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")

	currentExample := -1
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))

		if title, ok := parseTLDRTitle(trimmed); ok {
			if parsed.Title == "" {
				parsed.Title = title
			}
			continue
		}

		if desc, ok := parseTLDRDescription(trimmed); ok {
			parsed.Description = append(parsed.Description, desc)
			continue
		}

		if label, ok := parseTLDRExampleLabel(trimmed); ok {
			parsed.Examples = append(parsed.Examples, tldrExample{Label: label})
			currentExample = len(parsed.Examples) - 1
			inFence = false
			continue
		}

		if currentExample == -1 {
			continue
		}

		if isTLDRFence(trimmed) {
			inFence = !inFence
			continue
		}

		if inFence {
			parsed.Examples[currentExample].Code = append(parsed.Examples[currentExample].Code, strings.TrimSuffix(line, "\r"))
			continue
		}

		if code, ok := parseTLDRInlineCode(trimmed); ok {
			parsed.Examples[currentExample].Code = append(parsed.Examples[currentExample].Code, code)
		}
	}

	parsed.Markdown = renderTLDRMarkdown(parsed)
	return parsed
}

func renderTLDRMarkdown(parsed tldrTemplate) string {
	var out strings.Builder

	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		title = "Command"
	}
	out.WriteString("# ")
	out.WriteString(title)

	desc := trimTLDRLines(parsed.Description)
	if len(desc) > 0 {
		out.WriteString("\n\n")
		for i, line := range desc {
			if i > 0 {
				out.WriteString("\n")
			}
			out.WriteString(line)
		}
	}

	examples := compactTLDRExamples(parsed.Examples)
	if len(examples) == 0 {
		out.WriteString("\n")
		return out.String()
	}

	out.WriteString("\n\n## Examples\n")
	for _, ex := range examples {
		out.WriteString("\n- ")
		if ex.Label != "" {
			out.WriteString(ex.Label)
		} else {
			out.WriteString("Example")
		}

		if len(ex.Code) == 0 {
			continue
		}

		out.WriteString("\n\n```sh\n")
		for i, codeLine := range ex.Code {
			if i > 0 {
				out.WriteString("\n")
			}
			out.WriteString(codeLine)
		}
		out.WriteString("\n```\n")
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func parseTLDRTitle(line string) (string, bool) {
	if !strings.HasPrefix(line, "# ") {
		return "", false
	}
	title := strings.TrimSpace(line[2:])
	if title == "" {
		return "", false
	}
	return title, true
}

func parseTLDRDescription(line string) (string, bool) {
	if !strings.HasPrefix(line, ">") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, ">")), true
}

func parseTLDRExampleLabel(line string) (string, bool) {
	if !strings.HasPrefix(line, "-") {
		return "", false
	}
	label := strings.TrimSpace(strings.TrimPrefix(line, "-"))
	if label == "" {
		label = "Example"
	}
	return label, true
}

func parseTLDRInlineCode(line string) (string, bool) {
	if len(line) < 2 || !strings.HasPrefix(line, "`") || !strings.HasSuffix(line, "`") {
		return "", false
	}
	if isTLDRFence(line) {
		return "", false
	}
	code := strings.Trim(line, "`")
	code = strings.TrimSpace(code)
	if code == "" {
		return "", false
	}
	return code, true
}

func isTLDRFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

func trimTLDRLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	if start >= end {
		return nil
	}

	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		out = append(out, strings.TrimSpace(lines[i]))
	}

	return out
}

func compactTLDRExamples(examples []tldrExample) []tldrExample {
	out := make([]tldrExample, 0, len(examples))
	for _, ex := range examples {
		label := strings.TrimSpace(ex.Label)
		code := trimTLDRLines(ex.Code)
		if label == "" && len(code) == 0 {
			continue
		}
		out = append(out, tldrExample{Label: label, Code: code})
	}
	return out
}

func deriveTLDRCommand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}

	command := strings.TrimFunc(fields[0], func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' || r == '.')
	})
	return strings.ToLower(command)
}

func tldrPlatformTag(candidateID string) string {
	parts := strings.Split(candidateID, "/")
	if len(parts) < 2 {
		return ""
	}
	platform := strings.TrimSpace(parts[len(parts)-2])
	if platform == "" || strings.EqualFold(platform, "pages") || strings.EqualFold(platform, "common") {
		return ""
	}
	return strings.ToLower(platform)
}
