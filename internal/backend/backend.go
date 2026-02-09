package backend

import "cheatr/internal/backend/parsers"

type Backend interface {
	Init() error
	Update() error
	UpdateSource(name string) error

	Resolve(args []string) (*Resolution, error)
	GetEntry(res *Resolution) (*parsers.Entry, error)
	GetSection(res *Resolution) (string, error)

	Search(query string, filter SourceFilter) ([]SearchResult, error)

	ListDevDocs() ([]DevDoc, error)
	EnableDevDoc(slug string) error
	DisableDevDoc(slug string) error
}
