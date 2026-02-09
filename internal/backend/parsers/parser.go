package parsers

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
