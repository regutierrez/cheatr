# cheatr

Terminal-first programming cheatsheet aggregator.

## Developing

### Requirements

- Go 1.25+
- Internet access on first run (sources are downloaded into `~/.local/share/cheatr`)

### Run locally

```bash
go run ./cmd/cheatr
```

### Useful commands

```bash
# refresh all docs sources
go run ./cmd/cheatr update

# open DevDocs directly
go run ./cmd/cheatr docs python

# build the CLI binary
go build ./cmd/cheatr

# run tests
go test ./...
```

### Optional LLM setup

Create `~/.config/cheatr/config.yaml` to enable the LLM fallback in search/cascade flows.
