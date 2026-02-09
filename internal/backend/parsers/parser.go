package parsers

import (
	"context"
	"errors"
)

const (
	SourceLXIYM    = "lxiym"
	SourceDevhints = "devhints"
	SourceTldr     = "tldr"
	SourceDevDocs  = "devdocs"
)

const (
	CategorySyntax     = "syntax"
	CategoryCheatsheet = "cheatsheet"
	CategoryCommand    = "command"
	CategoryAPI        = "api"
)

type Entry struct {
	ID       string
	Source   string
	Topic    string
	Title    string
	Tags     []string
	Content  string
	Category string
	Sections []Section
}

type Section struct {
	Heading string
	Content string
}

type CandidateKind string

const (
	CandidateFile   CandidateKind = "file"
	CandidateRecord CandidateKind = "record"
)

type Candidate struct {
	Source  string
	Kind    CandidateKind
	ID      string
	Path    string
	Topic   string
	Title   string
	Payload string
	Meta    map[string]string
}

type DiscoverOptions struct {
	Root  string
	Limit int
}

type ParseOptions struct {
	Root string
}

type LoadOptions struct {
	Discover        DiscoverOptions
	Parse           ParseOptions
	ContinueOnError bool
}

type ParseFunc func(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error)

type Loader interface {
	Source() string
	Discover(ctx context.Context, opts DiscoverOptions) ([]Candidate, error)
	Parse(ctx context.Context, candidate Candidate, opts ParseOptions) (*Entry, error)
	Load(ctx context.Context, opts LoadOptions) ([]*Entry, error)
}

var ErrSkipCandidate = errors.New("skip candidate")
