package backend

import (
	"strings"
	"unicode"
)

var resolverAliasMap = map[string]string{
	"cplusplus": "cpp",
	"cxx":       "cpp",
	"js":        "javascript",
	"ts":        "typescript",
	"nodejs":    "node",
	"postgres":  "postgresql",
}

func normalizeResolverAlias(value string) string {
	key := normalizeResolverAliasKey(value)
	if key == "" {
		return ""
	}

	if alias, ok := resolverAliasMap[key]; ok {
		return alias
	}

	return key
}

func normalizeResolverAliasKey(value string) string {
	value = normalizeTopic(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == ' ', r == '-', r == '_', r == '.':
			continue
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}
