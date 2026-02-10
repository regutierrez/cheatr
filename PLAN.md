# Cheatr

A terminal-first programming cheatsheet aggregator that unifies multiple documentation sources into a single, searchable TUI.

## Data Sources

| Source | Content Type | Format | Access Method |
|---|---|---|---|
| [LearnXinYMinutes](https://github.com/adambard/learnxinyminutes-docs) | Language syntax walkthroughs | Markdown (YAML frontmatter) | Clone repo |
| [Devhints](https://github.com/rstacruz/cheatsheets) | Terse reference cards | Markdown (YAML frontmatter) | Clone repo |
| [tldr-pages](https://github.com/tldr-pages/tldr) | CLI command examples | Markdown (strict template) | Clone repo |
| [DevDocs](https://devdocs.io) | Full API documentation | HTML via JSON API | HTTP fetch (`documents.devdocs.io/{slug}/db.json`) |

## Architecture

Frontend and backend are fully separated. The backend is a Go package with a clean interface. The TUI imports it as a library. A future web frontend wraps the same package behind an HTTP server.

```
┌──────────────────────────────────────────────────────┐
│                     Frontends                        │
│                                                      │
│   ┌─────────────┐              ┌──────────────────┐  │
│   │  TUI (v1)   │              │  Web App (future) │  │
│   │  Bubble Tea │              │  HTTP server      │  │
│   │  + Glamour  │              │  wrapping backend │  │
│   └──────┬──────┘              └────────┬─────────┘  │
│          │                              │            │
└──────────┼──────────────────────────────┼────────────┘
           │ Go function calls            │ HTTP/JSON
           ▼                              ▼
┌──────────────────────────────────────────────────────┐
│                   Backend (Go pkg)                   │
│                                                      │
│   ┌────────────┐  ┌────────────┐  ┌───────────────┐  │
│   │   Sources  │  │   Index /  │  │   Content     │  │
│   │   Manager  │  │   Search   │  │   Renderer    │  │
│   └────────────┘  └────────────┘  └───────────────┘  │
│                                                      │
│   ┌────────────┐  ┌────────────┐  ┌───────────────┐  │
│   │   Cache    │  │   Parsers  │  │   Config      │  │
│   └────────────┘  └────────────┘  └───────────────┘  │
│                                                      │
└──────────────────────────────────────────────────────┘
```

## CLI Usage & Routing

The CLI uses positional args with smart routing to determine which source to query.

```
cheatr <topic> [subtopic]
cheatr update
```

### Routing Logic

```
cheatr <arg1> [arg2...]
  │
  ├─ Is arg1 a known programming language? (exists in LXIYM filenames)
  │   │
  │   ├─ YES, no subtopic ──▶ LearnXinYMinutes (full language overview)
  │   │     e.g. cheatr python
  │   │          cheatr rust
  │   │
  │   └─ YES, has subtopic ──▶ Cascade (see Subtopic Resolution below)
  │         e.g. cheatr python slices
  │              cheatr go goroutines
  │              cheatr js promises
  │
  └─ NO (not a programming language)
      │
      ├─ Exists in tldr? ──▶ tldr-pages (CLI command reference)
      │     e.g. cheatr docker
      │          cheatr curl
      │          cheatr tar
      │
      └─ Not in tldr? ──▶ Devhints (tool/framework cheatsheet)
            e.g. cheatr xpath
                 cheatr cron
```

### Subtopic Resolution (cascade)

When `cheatr <language> <subtopic>` is invoked, sources are tried in order until one hits:

```
cheatr python request
  │
  │  Step 1: Devhints
  ├─ Search for devhints match (file: python-request.md, or ### section match)
  │   └─ Found? ──▶ Render cheatsheet. Done.
  │
  │  Step 2: DevDocs (exact)
  ├─ Auto-download Python doc bundle if not cached
  ├─ Search "request" in Python doc index
  │   └─ Exact match? ──▶ Render doc page. Done.
  │
  │  Step 3: Selection list
  ├─ Fuzzy search "request" in Python doc index
  └─ Show selection list with all candidates + LLM option:
       ┌──────────────────────────────────────────────┐
       │  Results for "request" in python:             │
       │                                               │
       │  > urllib.request                              │
       │    http.client.HTTPRequest                     │
       │    requests.Session                            │
       │    ─────────────────────────────               │
       │    Ask llama3 (ollama)                         │
       └──────────────────────────────────────────────┘
       │
       ├─ User picks a doc entry ──▶ Render doc page. Done.
       └─ User picks LLM ──▶ Query LLM, render response. Done.
```

If there are zero DevDocs matches, the list still appears with only the LLM option:

```
       ┌──────────────────────────────────────────────┐
       │  No results for "request" in python.          │
       │                                               │
       │  > Ask llama3 (ollama)                         │
       └──────────────────────────────────────────────┘
```

### DevDocs Subcommand

DevDocs can also be accessed directly, bypassing the cascade:

```
cheatr docs <slug> [search]
  │
  ├─ Auto-download doc bundle if not cached
  │
  ├─ No search term ──▶ Browsable list of all entries (from index.json)
  │                      Select one → Glamour pager
  │
  ├─ Search term, exact match ──▶ Straight to Glamour pager
  │
  ├─ Search term, multiple matches ──▶ Filtered list → select → pager
  │
  └─ Search term, no matches ──▶ "No results for '<search>' in <slug> docs.
                                   Try: cheatr <slug> <search>"
```

### LLM Fallback

The LLM option appears in two places:
- As the last item in the Step 3 cascade selection list (`cheatr <language> <subtopic>`).
- As a no-results action in interactive search (`Ask <model> (<provider>)`).

It is:
- **Always visible**: shown as `Ask <model> (<provider>)` in the selection list, never fires without user selecting it
- **Provider-agnostic**: configurable in `~/.config/cheatr/config.yaml` (OpenAI, Ollama, etc.)
- **Context-aware**: sends the language + subtopic as the query, asks for a concise cheatsheet-style answer
- **Rendered the same way**: LLM response is treated as markdown, passed through Glamour
- **Hidden if disabled**: if `llm.enabled` is `false` or no config exists, the option doesn't appear in the list

```yaml
# ~/.config/cheatr/config.yaml
llm:
  enabled: true
  provider: ollama        # ollama | openai | anthropic
  model: llama3           # model name
  base_url: http://localhost:11434  # for ollama
  # api_key: ...          # for cloud providers (use env var CHEATR_API_KEY instead)
```

### Examples

| Command | Source | What you get |
|---|---|---|
| `cheatr python` | LearnXinYMinutes | Full Python syntax walkthrough |
| `cheatr rust` | LearnXinYMinutes | Full Rust syntax walkthrough |
| `cheatr python slices` | Devhints | Python slices cheatsheet |
| `cheatr python request` | Devhints → DevDocs → LLM | Cascade until found |
| `cheatr js promises` | Devhints | JavaScript promises reference |
| `cheatr docker` | tldr | Docker CLI command examples |
| `cheatr curl` | tldr | curl CLI usage |
| `cheatr tar` | tldr | tar CLI usage |
| `cheatr xpath` | Devhints | XPath cheatsheet (not in tldr) |
| `cheatr cron` | Devhints | Cron syntax reference |
| `cheatr docs python` | DevDocs | Browsable Python API docs |
| `cheatr docs python request` | DevDocs | Search "request" in Python docs |
| `cheatr update` | — | Pull latest from all sources |

### Language Detection

The set of known programming languages is derived from the filenames in the cloned LXIYM repo (e.g. `python.md` → `python`, `rust.md` → `rust`). This list is built at init time and cached. No hardcoded list needed.

## Backend Package (`internal/backend`)

### Source Manager

- Clones/updates Git repos (LXIYM, Devhints, tldr) to a local data directory (`~/.local/share/cheatr/`)
- Fetches DevDocs index from `https://devdocs.io/docs.json`
- Downloads DevDocs doc bundles (`db.json` + `index.json`) for user-selected docs
- `cheatr update` refreshes all sources

### Resolver (`internal/backend/resolver.go`)

The resolver implements the routing logic and cascade. It takes the CLI args and returns resolved content, trying sources in order.

```go
type Resolution struct {
    Source   string // "lxiym" | "devhints" | "tldr" | "devdocs" | "llm"
    Topic    string // primary topic (language or command name)
    Subtopic string // optional subtopic
    Content  string // resolved markdown content
    Candidates []Candidate // for DevDocs related matches (user picks one)
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
    Kind     SearchResultKind // entry | action
    Entry    *Entry           // set when Kind == SearchEntry
    Label    string           // set when Kind == SearchAction
    Source   string           // source badge for UI (e.g. "devdocs", "llm")
    Action   SearchActionKind // set when Kind == SearchAction
    Meta     map[string]string // optional action payload (query, slug, etc.)
    Score    int              // fuzzy match score (entries)
    Priority int              // routing-based rank (0 = highest)
}

// SourceFilter restricts search results to a single source.
// Empty string means no filter (show all sources, smart-ranked).
type SourceFilter string

const (
    FilterNone     SourceFilter = ""         // All sources (default)
    FilterLXIYM    SourceFilter = "lxiym"
    FilterDevhints SourceFilter = "devhints"
    FilterTldr     SourceFilter = "tldr"
    FilterDevDocs  SourceFilter = "devdocs"
)

type Resolver interface {
    Resolve(args []string) (*Resolution, error)
    ResolveSubtopic(lang, subtopic string) (*Resolution, error) // cascade logic
    ResolveDocs(slug string, search string) (*Resolution, error) // direct devdocs access
    Search(query string, filter SourceFilter) ([]SearchResult, error) // smart search with optional source filter + contextual action rows
    IsLanguage(name string) bool
    HasTldrEntry(name string) bool
    HasLocalDevDoc(slug string) bool
}
```

### LLM Client (`internal/backend/llm.go`)

Handles the optional LLM fallback:
- Reads provider config from `~/.config/cheatr/config.yaml`
- Sends a structured prompt: "Give a concise cheatsheet for <subtopic> in <language>"
- Returns markdown response
- Supports Ollama (local), OpenAI, Anthropic

### Parsers (per-source, `internal/backend/parsers/`)

Each parser normalizes content into a common type:

```go
type Entry struct {
    ID       string   // unique key, e.g. "lxiym:python", "devhints:docker"
    Source   string   // "lxiym" | "devhints" | "tldr" | "devdocs"
    Topic    string   // language or tool name
    Title    string   // display title
    Tags     []string // searchable tags
    Content  string   // markdown (DevDocs HTML converted via html-to-markdown)
    Category string   // "syntax" | "cheatsheet" | "command" | "api"
    Sections []Section // parsed subsections (for subtopic matching)
}

type Section struct {
    Heading string
    Content string
}
```

Parser specifics:
- **LXIYM**: Parse YAML frontmatter for metadata, body is already markdown. Category: `syntax`.
- **Devhints**: Parse YAML frontmatter, `###` sections parsed into `Sections` for subtopic lookup. Category: `cheatsheet`.
- **tldr**: Parse `# title`, `> description`, `- example` + code block structure. Category: `command`.
- **DevDocs**: Fetch `db.json`, convert HTML values to markdown using `html-to-markdown`. Category: `api`.

### Content Renderer

- Returns raw markdown for any entry (or a specific section of an entry)
- TUI frontend passes it through Glamour
- Web frontend can render it however it wants

### Cache

- Parsed entries cached to disk as JSON/gob to avoid re-parsing on every launch
- Language list cached alongside
- Invalidated when source repos are updated
- DevDocs bundles cached locally with TTL

## Backend Interface

```go
type Backend interface {
    // Source management
    Init() error                          // first-time setup, clone repos
    Update() error                        // pull latest from all sources
    UpdateSource(name string) error       // pull a single source

    // Resolution & lookup
    Resolve(args []string) (*Resolution, error)
    GetEntry(res *Resolution) (*Entry, error)
    GetSection(res *Resolution) (string, error) // returns markdown for a subtopic

    // Smart search (for interactive mode)
    Search(query string, filter SourceFilter) ([]SearchResult, error) // ranked by routing priority, filtered by source tab, may include contextual action rows

    // DevDocs-specific
    ListDevDocs() ([]DevDoc, error)
    EnableDevDoc(slug string) error
    DisableDevDoc(slug string) error
}
```

## TUI Frontend (`internal/tui`)

Built with the Charm ecosystem:
- **Bubble Tea**: application framework, event loop
- **Bubbles**: text input, list, viewport components
- **Lip Gloss**: layout and styling
- **Glamour**: markdown rendering in the viewport

### Modes

**1. Direct mode** (args provided): Resolve immediately, render result in a pager.
**2. Interactive mode** (no args): Full TUI with search and browse.

### Screen Mockups

UI is simple: list/search + viewer. No permanent shortcut bar.

#### Direct mode (`cheatr python`, `cheatr docker`, etc.)
- Full-screen pager, Glamour markdown.
- Header shows query + source badge.
- Scroll content (`j/k`, paging keys), `q` quits, `?` help.

#### Cascade selection (`cheatr <lang> <subtopic>`)
- If Devhints misses: show DevDocs candidates.
- Last row can be `Ask <model> (<provider>)` when enabled.
- Enter opens selected row.

#### Interactive mode (`cheatr`)
- Top: `Search` input.
- Tabs: `All -> LXIYM -> Devhints -> tldr -> DevDocs` (`Tab` forward).
- Ranking follows routing (language favors LXIYM, CLI favors tldr).
- Filter tab shows only that source.

DevDocs shortcuts in search:
- Inject `Browse <slug> docs (local)` when query exactly matches local slug/alias.
- Strict match only (normalized exact): no prefix, no fuzzy.
- Allowed in `All` and `DevDocs` tabs.

Alias normalization (strict):
- normalize: lowercase + strip `space`, `-`, `_`, `.`.
- explicit aliases only (examples):
  - `cplusplus`, `cxx` -> `cpp`
  - `js` -> `javascript`
  - `ts` -> `typescript`
  - `nodejs` -> `node`
  - `postgres` -> `postgresql`

No-result state:
- Show action rows:
  - `Search "<query>" in DevDocs`
  - `Ask <model> (<provider>)` (only when configured)

Viewer behavior in interactive:
- Pager style, no always-on hints.
- Mode badge: `SEARCH` / `LIST` / `VIEWER`.
- `?` toggles modal keyboard help (works even when search input focused).

Minimal no-result sketch:

```
Search: xyzabc█
[ All ] LXIYM Devhints tldr DevDocs
No local matches.
> Search "xyzabc" in DevDocs   [devdocs]
  Ask llama3 (ollama)          [llm]
```

Two baseline mockups (keep these while building):

```
Interactive search (list)
┌──────────────────────────────────────────────────────────┐
│ Search: python█                                          │
│ [ All ] LXIYM Devhints tldr DevDocs                      │
│ > python                   Learn Python          [lxiym] │
│   python-slices            Slices            [devhints]  │
│   python                   CLI usage             [tldr]  │
└──────────────────────────────────────────────────────────┘
```

```
Interactive viewer (pager)
┌──────────────────────────────────────────────────────────┐
│ cheatr [VIEWER] python slices                 [devhints] │
│ # Python Slices                                          │
│ ### Basic Slicing                                        │
│ a[1:3], a[:3], a[2:], a[-2:]                            │
│                                                    ▼ 42% │
└──────────────────────────────────────────────────────────┘
```

### Key Bindings

| Key | Context | Action |
|---|---|---|
| `Enter` | Interactive list screens | Open selected entry or execute selected action row |
| `Tab` | Interactive mode (non-input focus) | Cycle source filter forward (All → LXIYM → Devhints → tldr → DevDocs) |
| `Backspace` | Interactive mode (non-input focus) | Go back to previous selection screen; on root search results, cycle source filter backward |
| `/` | Interactive mode | Focus search input |
| `j/k` or `↑/↓` | Interactive list screens | Move selection up/down |
| `j/k` | Interactive article view (search not focused) | Scroll one line down/up |
| `f` / `b` | Interactive article view (search not focused) | Page down / page up |
| `g` / `G` | Interactive article view (search not focused) | Jump to top / bottom |
| `Esc` or `q` | Interactive mode | Exit program |
| `?` | Interactive mode (any focus, including search input) | Toggle keyboard help overlay |

### Rendering Pipeline

```
LXIYM markdown ──────┐
Devhints markdown ───┤
tldr markdown ───────┤──▶ Glamour.Render() ──▶ Viewport / Pager
DevDocs HTML ─▶ html-to-markdown ─┘
```

## Project Structure

```
cheatr/
├── cmd/
│   └── cheatr/
│       └── main.go              # CLI entrypoint, arg parsing, mode dispatch
├── internal/
│   ├── backend/
│   │   ├── backend.go           # Backend interface + implementation
│   │   ├── resolver.go          # Routing logic, cascade, source selection
│   │   ├── sources.go           # Source manager (clone, update)
│   │   ├── cache.go             # Disk cache
│   │   ├── llm.go               # LLM fallback client (ollama/openai/anthropic)
│   │   ├── config.go            # Config file parsing (~/.config/cheatr/config.yaml)
│   │   └── parsers/
│   │       ├── parser.go        # Common types (Entry, Section, etc.)
│   │       ├── lxiym.go         # LearnXinYMinutes parser
│   │       ├── devhints.go      # Devhints parser
│   │       ├── tldr.go          # tldr-pages parser
│   │       └── devdocs.go       # DevDocs fetcher + HTML-to-MD
│   └── tui/
│       ├── app.go               # Root Bubble Tea model, mode dispatch
│       ├── pager.go             # Direct mode: Glamour-rendered pager
│       ├── search.go            # Interactive mode: search screen
│       ├── viewer.go            # Interactive mode: entry viewer
│       └── styles.go            # Lip Gloss style definitions
├── go.mod
├── go.sum
├── PLAN.md
└── README.md
```

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | TUI components (textinput, list, viewport) |
| `github.com/charmbracelet/lipgloss` | TUI styling and layout |
| `github.com/charmbracelet/glamour` | Markdown → styled terminal output |
| `github.com/JohannesKaufmann/html-to-markdown` | DevDocs HTML → Markdown conversion |
| `github.com/go-git/go-git/v5` | Clone and pull source repos |
| `github.com/sahilm/fuzzy` | Fuzzy search matching |
| `gopkg.in/yaml.v3` | Parse YAML frontmatter |

## Implementation Plan (Todo)

Use this as the build checklist. Items are ordered by dependency.

### MVP (v1) -- DONE

1. **Scaffold project**: `go mod init`, create planned directories/files, wire `cmd/cheatr/main.go` mode dispatch.
2. **Define core types and interfaces**: `Entry`, `Section`, `Resolution`, `Candidate`, `SearchResult` (`entry` + `action` kinds), `SourceFilter`, `Backend`, and `Resolver`.
3. **Implement source manager**: clone/pull LXIYM, Devhints, tldr into `~/.local/share/cheatr/`; fetch DevDocs catalog (`docs.json`).
4. **Implement update commands**: `cheatr update` (all sources) and per-source update plumbing.
5. **Implement parser framework**: shared parser contracts and source-specific loaders.
6. **Implement LXIYM parser**: frontmatter + markdown body; language entries categorized as `syntax`.
7. **Implement Devhints parser**: frontmatter + `###` section extraction for subtopic lookup; category `cheatsheet`.
8. **Implement tldr parser**: title/description/examples template parsing; category `command`.
9. **Implement DevDocs data layer**: manage enabled docsets, download/cache `index.json` + `db.json` bundles per slug on demand, and convert HTML to markdown.
10. **Implement language detection**: derive known language set from LXIYM filenames and cache it.
11. **Implement top-level resolver routing**: language -> LXIYM, non-language -> tldr if present, otherwise Devhints.
12. **Implement subtopic cascade**: Devhints exact/section match -> DevDocs exact -> DevDocs related selection list.
13. **Implement direct DevDocs command**: `cheatr docs <slug> [search]` with list/exact/filtered/no-match behaviors.
14. **Implement content renderer**: return markdown for full entries/sections; use one rendering path for all sources.
15. **Implement direct pager mode**: full-screen Glamour viewport for arg-based commands (`cheatr python`, etc.).
16. **Implement smart search engine**: `Resolver.Search(query, filter)` with routing-priority ranking and source filtering.
17. **Implement source filter tabs**: All/LXIYM/Devhints/tldr/DevDocs with forward cycle on `Tab`.
18. **Implement interactive search UI**: search input + ranked result list + source grouping behavior.
19. **Implement strict local DevDocs injection**: add `Browse <slug> docs (local)` action row only for exact normalized local slug/alias matches.
20. **Implement alias normalization**: lowercase + separator stripping (`space`, `-`, `_`, `.`) and explicit alias map (e.g. `cplusplus`->`cpp`, `js`->`javascript`, `nodejs`->`node`).
21. **Implement no-results action rows**: show `Search "<query>" in DevDocs`; show `Ask <model> (<provider>)` only when LLM is configured.
22. **Implement action execution**: `Enter` opens entry rows and executes action rows.
23. **Implement interactive navigation model**: focus states (`SEARCH`/`LIST`/`VIEWER`) and contextual `Backspace` behavior.
24. **Implement interactive viewer behavior**: pager-style reading in interactive mode with `j/k` line scroll, `f/b` page scroll, `g/G` top/bottom.
25. **Implement global help overlay**: toggle with `?` from any focus (including search input), close with `?`/`Esc`/`q`; no always-on key hint strip.
26. **Implement exit/focus shortcuts**: `/` refocus search input; `Esc`/`q` quit interactive mode.
27. **Implement cache layer**: parsed-entry cache, language list cache, invalidation on source updates, DevDocs bundle TTL handling.

### BUGFIXES/IMPROVEMENTS
- polish TUI. it's not good currently. draw it out probably?
  - straighten out keyboard navigation. (/ for search text should be a thing. remap search text to other key maybe?)
  - probably use learnxinyminutes primary resource. use devhints as fallback if nothing pops up in lxiym (claude-code, tmux cheatsheets, etc.)
    need a better way to parse sections in learnxinyminutes if that's the case. or scroll into that part of the page already?
  - add % label at bottom right
- results must be paginated.
- better fuzzy-finding-- strict enough that results closely follow search text, loose enough for uppercase/lowercase, list vs. lists
  - probably add keywords also (for loops can be iterables. if statements can be conditionals. so on and so forth)
- how to handle devdocs results since they're a lot.
- for first time run, make sure to grab the docs we need. and only grab stuff that we need (english only, etc.)
- add way to add personal cheatsheets.


### Future

28. **Config system**: parse `~/.config/cheatr/config.yaml` with provider settings.
29. **LLM client**: provider abstraction (Ollama/OpenAI/Anthropic) and structured markdown responses.
30. **LLM in cascade list**: append `Ask <model> (<provider>)` as last selectable item in Step 3 selection UI.
31. **Interactive DevDocs browser (in-TUI)**: dedicated docset picker -> entry list -> viewer flow with preserved back-navigation state.
32. **Web frontend**: HTTP server wrapping the backend interface.
