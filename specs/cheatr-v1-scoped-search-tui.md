# Cheatr v1 — Scoped Search TUI Spec

**Status:** Ready for task breakdown
**Type:** Feature plan
**Date:** 2026-03-31
**Effort:** XL
**Approved by:** user (conversation)

## Problem Statement

**Who:** terminal-first developers who want syntax or command reminders without context-switching into a browser.

**What:** they want to select a scope, type a short query, and get an immediate, local answer card or command page. Example intents include Python syntax queries like `list`, `append list`, `dictionary`, and command queries like `find name` or `tar extract`.

**Why it matters:** the original aggregator-first design optimizes for breadth, but the user goal is speed, determinism, and clarity. If v1 feels like a docs browser instead of an answer finder, it misses the brief.

**Evidence:** explicit user feedback in planning conversation:

- search should be scoped before the query
- answers should be instantaneous
- start small and extensible
- no model in v1
- no source tabs
- include terminal commands in a tldr-like way

## Recommendation

Build a **scoped local search TUI** with exactly two v1 scopes:

- `Python`
- `Commands`

The app should never search across scopes implicitly. The user selects a scope first, then types a query.

### Python scope

Python content is authored as curated local markdown concept cards, one file per concept. Each card is human-editable and display-ready, but it is **not** the runtime search structure. A **qualification** step first checks that the page is eligible for the corpus (schema, provenance, self-contained snippets, and Python parseability). A later compile step extracts canonical sections, normalizes metadata, and emits a locked runtime index.

### Commands scope

Commands content is seeded from a bundled snapshot of `tldr` pages (`common + linux` only), normalized into the same runtime page model used by Python cards, and optionally refreshed later via `cheatr update`. Commands are trusted upstream content in v1, so they are structurally qualified for importability and overrides, but they are **not** execution-verified.

### References

Python pages link out to DevDocs. Command pages link to an official reference URL if present in local metadata; otherwise they link to the upstream tldr page.

This keeps the primary path fast and deterministic while preserving an external escape hatch.

## Scope & Deliverables

| Deliverable                                                                          | Effort | Depends On |
| ------------------------------------------------------------------------------------ | ------ | ---------- |
| D0. Finalize pilot Python concept inventory and page IDs                             | M      | -          |
| D1. Define unified runtime page schema + compiled index format                       | M      | D0         |
| D2. Implement Python card authoring format, parser, and qualification checks         | M      | D1         |
| D3. Implement Commands snapshot import, normalization, qualification, and overrides  | L      | D1         |
| D4. Implement scoped search, normalization, and ranking                              | M      | D2, D3     |
| D5. Implement TUI flows for scope switch, search list, viewer, and no-result actions | L      | D4         |
| D6. Implement persisted UI state + stale-data prompt + `cheatr update`               | M      | D3, D5     |
| D7. Acceptance tests, golden query fixtures, and polish                              | M      | D4, D5, D6 |

## Implementation Phases

### Phase 0 — Content Boundary

- Finalize the initial Python pilot concept set.
- Assign stable page IDs (`python/list`, `python/dict`, etc.).
- Define the DevDocs slug/search target for Python.
- Define the global alias table and Python-specific alias table.

### Phase 1 — Content Compiler Foundation

- Introduce the unified runtime `Page` and `Section` schema.
- Introduce a separate corpus qualification pass before compilation.
- Implement compilation from authored markdown and imported command pages into one indexed format.
- Add qualification and compilation boundaries so invalid content never reaches the runtime index.

### Phase 2 — Python Content Pipeline

- Add one markdown file per Python concept.
- Parse frontmatter, required syntax block, and `##` section structure.
- Qualify Python pages using the minimal corpus-admission checks defined below.
- Compile qualified pages into a runtime index.

### Phase 3 — Commands Content Pipeline

- Bundle a raw tldr snapshot for `common + linux`.
- Normalize tldr pages into runtime pages.
- Qualify imported command content for importability and override consistency only.
- Add optional local overrides for aliases, keywords, labels, and reference URLs.
- Persist `last_updated` metadata.

### Phase 4 — Search + Ranking

- Build scoped search over the compiled index.
- Apply deterministic matching buckets first.
- Use fuzzy scoring only as a tiebreaker.
- Enforce max 15 result rows, one row per page.

### Phase 5 — TUI UX

- Add top chips: `Python | Commands`.
- Preserve query when switching scope and immediately rerun search in the new scope.
- Open the selected page at the best-matching section/topic.
- Preserve query and scope when returning from the viewer.
- Persist last-used scope and last query across runs.

### Phase 6 — Update + Polish

- Implement `cheatr update` to refresh Commands data and rebuild the index.
- On startup, if Commands data is stale, show a prompt to update.
- Add no-result actions, browse-all flows, and golden query tests.

## Non-Goals (Explicit Exclusions)

- No LLM, embeddings, or semantic model-assisted runtime search in v1.
- No source tabs (`LXIYM`, `Devhints`, `tldr`, `DevDocs`).
- No unified cross-scope search in v1.
- No local DevDocs bundle ingestion or in-TUI docs browser in v1.
- No typo-tolerance in v1.
- No execution verification of imported Commands content in v1.
- No web frontend in v1.
- No shell-script/Bash syntax scope in v1; `Commands` means external commands only.
- No additional language scopes in v1 beyond Python.

## Product UX

### Scope Model

Visible top-level chips:

- `Python`
- `Commands`

Rules:

- scope must be selected before the query is interpreted
- search is strictly scoped to the selected scope
- query is preserved when switching scopes and rerun immediately
- last-used scope and last query are restored on startup

### Empty States

- **Python:** show all Python concepts alphabetically.
- **Commands:** show the first N commands alphabetically from the imported dataset.

### Browse-All

Browse-all uses the same list UI as search, but unfiltered.

- Python: `Browse all concepts`
- Commands: `Browse commands`

### Result Rows

- one row per page only
- row format: `Title — label`
- `label` is the best-matching section for Python or topic label for Commands
- action rows do **not** count toward the 15-result cap

Examples:

- `List — append`
- `Dict — syntax`
- `find — name`
- `tar — extract`

### Viewer Behavior

- `Enter` opens the selected page.
- `Backspace`/`Esc` from the viewer returns to the search list with scope and query preserved.
- pages open with a pinned top block, then jump to the best-matching section/topic.

#### Python viewer top block

Always present:

- title
- verified version
- required syntax block
- `Open docs`

#### Commands viewer top block

Always present:

- command title
- summary
- `Open reference`

### No-Results Behavior

#### Python

Show an explicit local miss state, then offer a docs action:

- preferred: `Search "<query>" in docs`
- fallback: `Open docs`

Use the search variant only when a real query deep-link can be constructed for the selected docs target.

#### Commands

Show an explicit local miss state, then offer:

- `Browse commands`

## Content Strategy

### Authoring vs Runtime (Hybrid Model)

Author markdown is the **reviewable source format**, not the final runtime structure.

The pipeline is intentionally hybrid:

1. humans author or review readable markdown pages
2. a compiler validates and normalizes them
3. the TUI queries only the compiled runtime index

This achieves:

- human-friendly editing and review
- deterministic runtime behavior
- schema enforcement
- faster search than scanning raw markdown at query time

### Qualification vs Compilation

These are separate stages.

- **Qualification** answers: "is this source content eligible to join the corpus at all?"
- **Compilation** answers: "turn already-qualified content into the locked runtime index used by the TUI."

Qualification is the admission gate for authored Python content and imported Commands content.

#### Python qualification (v1 minimal policy)

Python uses the lightweight Option A policy. A Python page is corpus-eligible when it passes all of the following checks:

- required frontmatter is present
- required DevDocs target is present
- required top syntax block is present
- file path, `id`, `scope`, and title are consistent
- `##` section structure is valid
- all Python code fences use the `python` language tag
- snippets are self-contained enough to read honestly; if stdlib is needed, the snippet must import it explicitly
- all Python code blocks parse successfully with `ast.parse`
- related page IDs resolve
- normalized aliases do not collide dangerously with other Python pages

What this does **not** guarantee:

- conceptual correctness of every explanation
- idiomatic quality of every snippet
- runtime behavior beyond syntax validity

This is intentionally a corpus-worthiness gate, not a full proof of truth.

#### Commands qualification (v1)

Commands are trusted from the bundled/imported `tldr` corpus. Qualification for Commands is limited to:

- page importability
- override schema validity
- normalized ID uniqueness
- valid reference URLs when overrides provide them

Commands are **not** execution-verified in v1.

## Data Model

### Unified Runtime Model

All content compiles into one runtime `Page` schema.

```go
type Scope string

const (
    ScopePython   Scope = "python"
    ScopeCommands Scope = "commands"
)

type Page struct {
    ID            string
    Scope         Scope
    Title         string
    Summary       string
    Aliases       []string
    Keywords      []string
    Version       string   // required for Python, empty for Commands
    DocsURL       string   // Python DevDocs target; empty if none
    ReferenceURL  string   // Commands official ref if present, else upstream tldr URL
    Related       []string // page IDs, mostly used by Python
    Sections      []Section
    Provenance    Provenance
}

type Section struct {
    ID        string
    Title     string
    Label     string   // short label used in result rows
    Aliases   []string
    Keywords  []string
    Body      string   // markdown
}

type Provenance struct {
    Kind       string // "authored" | "tldr"
    Source     string // file path or upstream identifier
    UpdatedAt  string // ISO-8601 when compiled/imported
}

type SearchResult struct {
    PageID        string
    Title         string
    Label         string
    SectionID     string
    Bucket        int
    FuzzyScore    int
}

type ActionRow struct {
    Kind   string // "browse" | "open_docs" | "search_docs"
    Label  string
    Target string
}
```

### Persisted UI State

```go
type UIState struct {
    Scope Scope
    Query string
}
```

### Compiled Index Contract

The compiled index must contain:

- all pages
- pre-normalized search fields
- precomputed aliases and keyword tokens
- search-ready section/topic metadata
- content version/hash used to detect rebuild necessity

## Python Authoring Format

### File Layout

One file per concept:

- `content/python/<slug>.md`

Examples:

- `content/python/list.md`
- `content/python/dict.md`
- `content/python/comprehension.md`

### Required Frontmatter

```yaml
id: python/list
scope: python
title: List
aliases: [lists, array]
keywords: [sequence, mutable, ordered]
version_verified: "Python 3.13"
docs_url: "https://devdocs.io/python~3.13/"
related: [python/tuple, python/set, python/dict]
```

### Required Body Shape

Rules:

- one `# <Title>` heading
- one required fenced code block immediately after the title; this is the syntax summary shown in the viewer top block
- at least one `##` section is required after the top syntax block
- only `##` headings are indexed as canonical searchable sections
- `###` headings are display-only in v1 and not first-class search targets
- pages may include any subset of the recommended canonical sections; missing sections are simply absent from the compiled index

Reference examples:

- full example with all recommended sections: `specs/python-corpus-samples.md`
- minimal valid example with only a subset of sections: `specs/python-corpus-samples.md`

Example shape:

````md
---
id: python/list
scope: python
title: List
aliases: [lists, array]
keywords: [sequence, mutable, ordered]
version_verified: "Python 3.13"
docs_url: "https://devdocs.io/python~3.13/"
related: [python/tuple, python/set, python/dict]
---

# List

```python
items = []
items = [1, 2, 3]
items = list(iterable)
```
````

## Syntax

...

## Create

...

## Access

...

## Modify

...

## Iterate

...

## Notes

...

## Pitfalls

...

## Related

...

### Recommended Section Order

Recommended, not strictly mandatory, section order:

1. `Syntax`
2. `Create`
3. `Access`
4. `Modify`
5. `Iterate`
6. `Notes`
7. `Pitfalls`
8. `Related`

Compiler rules:

- fail if frontmatter is missing required fields
- fail if syntax block is missing
- fail if there are zero `##` sections
- fail if `id`, `scope`, and `title` are inconsistent with file path
- parse `##` headings into runtime `Section` values
- derive default section `label` from heading text if no override exists

## Commands Source Format

### Raw Source

Bundle raw tldr pages for:

- `common`
- `linux`

Repository layout:

- `vendor/tldr/pages/common/*.md`
- `vendor/tldr/pages/linux/*.md`

### Runtime Strategy

The app ships with a bundled snapshot for first run. At runtime it may also maintain a user-local refreshed snapshot. The compiler prefers:

1. user-local updated snapshot, if present
2. bundled snapshot otherwise

### Local Override Layer

To support hybrid import + curation, allow local override files, e.g.:

- `content/commands/overrides/<command>.yaml`

Override fields may include:

- aliases
- keywords
- summary override
- preferred topic labels
- reference URL override
- hide/rename noisy topics

### Normalized Commands Page Shape

Commands pages must compile into the same `Page` schema with:

- title
- aliases
- summary
- topic labels
- examples/body sections
- reference URL
- provenance

### Topic Label Extraction

Command result-row labels are derived from the best matching imported topic, using this priority:

1. short normalized label override, if present
2. extracted flag/option label (`-r`, `-name`, `--extract`, etc.)
3. normalized example description

The row still represents one page only.

## Search & Ranking

### Query Normalization

Apply the following before matching:

- lowercase
- split on whitespace and punctuation
- singular/plural folding
- global alias expansion
- scope-specific alias expansion
- filler-word demotion, not removal

Examples of filler words to demote:

- `how`
- `to`
- `syntax`
- `make`
- `create`
- `declare`

Not in v1:

- typo tolerance
- embedding search
- LLM query rewriting

### Scope-Specific Search

Search candidates must come only from the selected scope.

### Ranking Algorithm

Apply deterministic buckets first, then fuzzy as a tiebreaker.

Bucket order:

1. exact normalized title or alias match
2. title/concept match
3. section/topic title match
4. keyword/body match

Tie-break rules within a bucket:

1. better fuzzy score
2. shorter/more exact title or alias match
3. alphabetical order

Additional rules:

- max 15 page rows returned
- max 1 row per page
- the row label is the best matching section/topic for that page
- action rows are appended separately and do not consume result slots
- query token order is treated as order-insensitive in v1

### Golden Query Expectations

Python:

- `list` → `List — syntax` or `List — create`
- `append list` → `List — append` or `List — modify`
- `dictionary` → `Dict — syntax` or `Dict — create`
- `dict comprehension` → whichever page has the stronger title/section match

Commands:

- `find name` → `find — name`
- `tar extract` → `tar — extract`
- `grep recursive` → `grep — recursive`

## Update & Freshness

### `cheatr update`

Responsibilities:

- refresh Commands snapshot from upstream tldr
- rebuild compiled index
- persist last-updated metadata

Python authored content is not remotely refreshed by `cheatr update` in v1.

### Startup Freshness Prompt

If Commands data is older than the configured stale threshold, the TUI should show a prompt to update.

Recommended default stale threshold: **30 days**.

This prompt must be non-blocking.

## API / Interface Contracts

### Qualifier

```go
type Qualifier interface {
    QualifyPython() (*QualificationReport, error)
    QualifyCommands() (*QualificationReport, error)
}

type QualificationReport struct {
    Errors   []string
    Warnings []string
}
```

### Compiler

```go
type Compiler interface {
    Compile() (*CompiledIndex, error)
    NeedsRebuild() (bool, error)
}
```

### Search

```go
type Searcher interface {
    Search(scope Scope, query string) ([]SearchResult, []ActionRow, error)
    Browse(scope Scope) ([]SearchResult, error)
}
```

### Update

```go
type Updater interface {
    UpdateCommands() error
    LastUpdated() (time.Time, error)
}
```

### State

```go
type StateStore interface {
    Load() (UIState, error)
    Save(UIState) error
}
```

## Proposed File / Module Layout

```text
cheatr/
├── content/
│   ├── python/
│   │   └── *.md
│   └── commands/
│       └── overrides/
│           └── *.yaml
├── vendor/
│   └── tldr/
│       └── pages/
│           ├── common/
│           └── linux/
├── internal/
│   ├── content/
│   │   ├── page.go
│   │   ├── qualify.go
│   │   ├── python_parser.go
│   │   ├── tldr_importer.go
│   │   ├── overrides.go
│   │   ├── compiler.go
│   │   └── validate.go
│   ├── search/
│   │   ├── normalize.go
│   │   ├── index.go
│   │   └── rank.go
│   ├── state/
│   │   └── store.go
│   ├── update/
│   │   └── commands.go
│   └── tui/
│       ├── app.go
│       ├── search.go
│       ├── viewer.go
│       └── styles.go
└── specs/
    └── cheatr-v1-scoped-search-tui.md
```

## Acceptance Criteria

- [ ] On startup, the app restores the last-used scope and last query.
- [ ] The top-level UX uses explicit scope chips: `Python | Commands`.
- [ ] Query interpretation is strictly scoped to the selected scope.
- [ ] Python pages are authored as one markdown file per concept and must pass qualification before they are eligible for compilation.
- [ ] Python qualification enforces required metadata, required syntax block, valid `##` section structure, explicit stdlib imports when needed, and successful `ast.parse` for every Python code block.
- [ ] Only `##` headings are indexed as canonical searchable sections in Python pages.
- [ ] Commands data works offline on first run from a bundled snapshot.
- [ ] Commands qualification is limited to importability/override validity and does not execute imported examples.
- [ ] `cheatr update` refreshes Commands data, rebuilds the index, and records last-updated metadata.
- [ ] The runtime search queries a compiled index rather than raw markdown files.
- [ ] Search returns at most 15 page rows and at most one row per page.
- [ ] Results use deterministic ranking buckets with fuzzy only as a tiebreaker.
- [ ] Python no-results state explicitly indicates no local result and offers a docs action.
- [ ] Commands no-results state explicitly indicates no local result and offers `Browse commands`.
- [ ] Enter opens the selected page at the best matching section/topic.
- [ ] Returning from the viewer restores the prior search state.
- [ ] Python pages expose `Open docs` to DevDocs.
- [ ] Commands pages expose `Open reference` to an official reference URL when present, otherwise upstream tldr.

## Test Strategy

| Layer       | What               | How                                                                                                    |
| ----------- | ------------------ | ------------------------------------------------------------------------------------------------------ |
| Unit        | Python qualifier   | validate frontmatter, syntax block, `##` section extraction, explicit imports, and `ast.parse` success |
| Unit        | Commands qualifier | validate tldr importability, override schema, and reference URL validity                               |
| Unit        | Python card parser | parse already-qualified pages into runtime page structures                                             |
| Unit        | Commands importer  | parse tldr pages, derive topic labels, apply overrides                                                 |
| Unit        | Normalization      | verify alias expansion, plural folding, filler demotion                                                |
| Unit        | Ranking            | golden-query fixtures for bucket ordering and tie-break behavior                                       |
| Integration | Qualification      | invalid Python pages fail corpus admission; valid Commands pages import without execution verification |
| Integration | Compiler           | qualified Python + bundled Commands compile into one runtime index                                     |
| Integration | Search             | scope-limited searches return correct page + label                                                     |
| Integration | TUI state          | switching scope preserves query and reruns search                                                      |
| Integration | Viewer             | opening a result jumps to expected section/topic and back restores search state                        |
| Integration | Update             | refreshed Commands data rebuilds index and updates freshness metadata                                  |

## Risks & Mitigations

| Risk                                            | Likelihood | Impact | Mitigation                                                                                                          |
| ----------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------------------------------------------------------- |
| Commands import produces noisy labels           | Medium     | Medium | support local overrides for labels, aliases, and hidden topics                                                      |
| Python card authoring drifts in structure       | Medium     | High   | enforce qualification checks, required syntax block, and `ast.parse` on all Python snippets                         |
| Search feels too brittle without typo tolerance | Medium     | Medium | lean on alias tables and filler demotion in v1; revisit typo tolerance after real usage                             |
| DevDocs deep-link search may be inconsistent    | Medium     | Low    | support `Search docs` only when a valid query URL exists; otherwise fall back to `Open docs`                        |
| Scope creep reintroduces aggregator behavior    | High       | High   | keep non-goals explicit and defer extra languages/local docs ingestion until after Python + Commands feel excellent |

## Trade-offs Made

| Chose                                    | Over                           | Because                                                                    |
| ---------------------------------------- | ------------------------------ | -------------------------------------------------------------------------- |
| Scoped search first                      | Unified all-content search     | explicit scope keeps ranking deterministic and UX understandable           |
| Curated Python cards                     | Runtime docs parsing           | local cards are faster, clearer, and easier to validate                    |
| Bundled tldr snapshot + optional refresh | Mandatory network fetch        | first run must work offline and feel immediate                             |
| Python qualification only                | Full execution verification    | catches structural and obvious snippet issues without complicating v1      |
| Trust imported tldr commands             | Verifying command examples     | command execution adds risk and complexity with limited value in v1        |
| Markdown authoring + compiled index      | Querying raw markdown directly | human-friendly editing plus locked runtime structure                       |
| DevDocs as external docs target          | Local DevDocs ingestion        | escape hatch is enough for v1; local docs browser is over-scope            |
| One page row per result                  | Multiple section rows per page | avoids flooding the list and keeps scan time low                           |
| No typo tolerance in v1                  | More forgiving fuzzy search    | deterministic results are more important than breadth in the first release |

## Success Metrics

- User can answer common Python syntax queries in 1-2 keystrokes after selecting Python.
- User can answer common command usage queries without leaving the TUI.
- First-run experience works offline for local content and bundled Commands data.
- No-result states clearly distinguish “not in local corpus” from “app failed.”
- Adding a future language requires only: new cards, a DevDocs mapping, and a per-language alias table.

## Rollback / Revisit Triggers

Revisit major decisions if:

- users consistently miss results due to absent typo tolerance
- one-row-per-page hides too much useful specificity
- DevDocs links are insufficient and an in-TUI docs browser becomes a recurring need
- adding a second language reveals schema gaps in the current unified model

## Ready-to-Build Summary

This spec replaces the original aggregator-first plan with a narrower v1:

- one pilot language: Python
- one command scope: Commands
- explicit scope chips before query
- local curated Python cards
- bundled tldr-derived Commands content
- compiled unified index
- deterministic search
- external docs/reference escape hatches
