package rules

import (
	"strings"

	"lucid-ci/engine/graph"
)

type Category string

const (
	CategorySource    Category = "source"
	CategorySink      Category = "sink"
	CategorySanitizer Category = "sanitizer"
)

type SinkKind string

const (
	SinkSQL     SinkKind = "sql"
	SinkCommand SinkKind = "command"
	SinkFile    SinkKind = "file"
)

type CatalogEntry struct {
	Name     string
	Category Category
	SinkKind SinkKind
	Language string
}

type Catalog struct {
	entries []CatalogEntry
}

func NewCatalog() Catalog {
	entries := make([]CatalogEntry, 0, len(jsEntries)+len(pyEntries))
	entries = append(entries, jsEntries...)
	entries = append(entries, pyEntries...)
	return Catalog{entries: entries}
}

func (c Catalog) IsSource(n graph.Node) bool {
	return c.match(n, CategorySource, "")
}

func (c Catalog) IsSink(n graph.Node) bool {
	return c.match(n, CategorySink, "")
}

func (c Catalog) SinkKind(n graph.Node) (SinkKind, bool) {
	name := canonicalName(n.Name)
	for _, entry := range c.entries {
		if entry.Category == CategorySink && nameMatches(name, entry.Name) {
			return entry.SinkKind, true
		}
	}
	return "", false
}

func (c Catalog) IsSanitizer(n graph.Node) bool {
	return c.match(n, CategorySanitizer, "")
}

func (c Catalog) ContainsSource(text string) bool {
	return c.contains(text, CategorySource, "")
}

func (c Catalog) ContainsSink(text string) bool {
	return c.contains(text, CategorySink, "")
}

func (c Catalog) ContainsSanitizer(text string) bool {
	return c.contains(text, CategorySanitizer, "")
}

func (c Catalog) Entries() []CatalogEntry {
	return append([]CatalogEntry(nil), c.entries...)
}

func (c Catalog) match(n graph.Node, category Category, sinkKind SinkKind) bool {
	name := canonicalName(n.Name)
	if name == "" {
		name = canonicalName(n.CodeSnippet)
	}
	for _, entry := range c.entries {
		if entry.Category != category {
			continue
		}
		if sinkKind != "" && entry.SinkKind != sinkKind {
			continue
		}
		if nameMatches(name, entry.Name) {
			return true
		}
	}
	return false
}

func (c Catalog) contains(text string, category Category, sinkKind SinkKind) bool {
	text = canonicalName(text)
	if text == "" {
		return false
	}
	for _, entry := range c.entries {
		if entry.Category != category {
			continue
		}
		if sinkKind != "" && entry.SinkKind != sinkKind {
			continue
		}
		name := canonicalName(entry.Name)
		if strings.Contains(text, name) {
			return true
		}
	}
	return false
}

func canonicalName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "()")
	name = strings.ReplaceAll(name, "?.", ".")
	return name
}

func nameMatches(actual, expected string) bool {
	actual = canonicalName(actual)
	expected = canonicalName(expected)
	if actual == expected {
		return true
	}
	if strings.HasPrefix(actual, expected+".") {
		return true
	}
	if strings.HasSuffix(actual, "."+expected) {
		return true
	}
	return strings.HasPrefix(actual, expected+"(")
}
