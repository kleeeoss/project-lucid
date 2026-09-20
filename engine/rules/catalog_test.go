package rules

import (
	"testing"

	"lucid-ci/engine/graph"
)

func TestCatalogIdentifiesSources(t *testing.T) {
	catalog := NewCatalog()
	for _, name := range []string{"req.query", "req.body", "request.args", "request.POST"} {
		if !catalog.IsSource(graph.Node{Name: name}) {
			t.Fatalf("expected %s to be source", name)
		}
	}
	if catalog.IsSource(graph.Node{Name: "safeValue"}) {
		t.Fatalf("safeValue must not be source")
	}
}

func TestCatalogIdentifiesSinks(t *testing.T) {
	catalog := NewCatalog()
	cases := map[string]SinkKind{
		"db.query":           SinkSQL,
		"cursor.execute":     SinkSQL,
		"child_process.exec": SinkCommand,
		"os.system":          SinkCommand,
		"fs.readFile":        SinkFile,
		"open":               SinkFile,
	}
	for name, want := range cases {
		node := graph.Node{Name: name}
		if !catalog.IsSink(node) {
			t.Fatalf("expected %s to be sink", name)
		}
		got, ok := catalog.SinkKind(node)
		if !ok || got != want {
			t.Fatalf("sink kind for %s = %q,%t want %q,true", name, got, ok, want)
		}
	}
	if catalog.IsSink(graph.Node{Name: "console.log"}) {
		t.Fatalf("console.log must not be sink")
	}
}

func TestCatalogIdentifiesSanitizers(t *testing.T) {
	catalog := NewCatalog()
	for _, name := range []string{"escape", "sanitize", "path.resolve", "os.path.abspath"} {
		if !catalog.IsSanitizer(graph.Node{Name: name}) {
			t.Fatalf("expected %s to be sanitizer", name)
		}
	}
	if catalog.IsSanitizer(graph.Node{Name: "identity"}) {
		t.Fatalf("identity must not be sanitizer")
	}
}

func TestCatalogDetectsSignaturesInCodeText(t *testing.T) {
	catalog := NewCatalog()
	if !catalog.ContainsSource("const id = req.query.id") {
		t.Fatalf("expected source signature inside JS assignment")
	}
	if !catalog.ContainsSink("cursor.execute('SELECT ' + name)") {
		t.Fatalf("expected SQL sink signature inside Python call")
	}
	if !catalog.ContainsSanitizer("const safe = sanitize(input)") {
		t.Fatalf("expected sanitizer signature inside assignment")
	}
	if catalog.ContainsSource("const id = user.id") {
		t.Fatalf("clean code must not contain source signature")
	}
}

func TestCatalogEntriesDefensiveCopy(t *testing.T) {
	catalog := NewCatalog()
	entries := catalog.Entries()
	if len(entries) == 0 {
		t.Fatalf("expected catalog entries")
	}
	entries[0].Name = "mutated"
	if catalog.Entries()[0].Name == "mutated" {
		t.Fatalf("Entries must return a defensive copy")
	}
}
