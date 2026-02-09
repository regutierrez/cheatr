package backend

import "cheatr/internal/backend/parsers"

const (
	SourceLXIYM    = parsers.SourceLXIYM
	SourceDevhints = parsers.SourceDevhints
	SourceTldr     = parsers.SourceTldr
	SourceDevDocs  = parsers.SourceDevDocs
	SourceLLM      = "llm"
)

type Resolution struct {
	Source     string
	Topic      string
	Subtopic   string
	Content    string
	Candidates []Candidate
}

type Candidate struct {
	Title string
	Path  string
}

type SearchResultKind string

const (
	SearchEntry  SearchResultKind = "entry"
	SearchAction SearchResultKind = "action"
)

type SearchActionKind string

const (
	ActionNone          SearchActionKind = ""
	ActionBrowseDevDocs SearchActionKind = "browse_devdocs"
	ActionAskLLM        SearchActionKind = "ask_llm"
)

type SearchResult struct {
	Kind     SearchResultKind
	Entry    *parsers.Entry
	Label    string
	Source   string
	Action   SearchActionKind
	Meta     map[string]string
	Score    int
	Priority int
}

type SourceFilter string

const (
	FilterNone     SourceFilter = ""
	FilterLXIYM    SourceFilter = SourceLXIYM
	FilterDevhints SourceFilter = SourceDevhints
	FilterTldr     SourceFilter = SourceTldr
	FilterDevDocs  SourceFilter = SourceDevDocs
)

type LanguageDetector interface {
	KnownLanguages() ([]string, error)
	IsLanguage(name string) bool
}

type Resolver interface {
	Resolve(args []string) (*Resolution, error)
	ResolveSubtopic(lang, subtopic string) (*Resolution, error)
	ResolveDocs(slug, search string) (*Resolution, error)
	Search(query string, filter SourceFilter) ([]SearchResult, error)
	IsLanguage(name string) bool
	HasTldrEntry(name string) bool
	HasLocalDevDoc(slug string) bool
}
