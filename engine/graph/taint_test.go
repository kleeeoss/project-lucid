package graph

import "testing"

type testCatalog struct{}

func (testCatalog) IsSource(n Node) bool            { return containsAny(n.Name, "req.query", "request.args") }
func (testCatalog) IsSink(n Node) bool              { return containsAny(n.Name, "db.query", "cursor.execute") }
func (testCatalog) IsSanitizer(n Node) bool         { return containsAny(n.Name, "sanitize", "escape") }
func (testCatalog) ContainsSource(s string) bool    { return containsAny(s, "req.query", "request.args") }
func (testCatalog) ContainsSink(s string) bool      { return containsAny(s, "db.query", "cursor.execute") }
func (testCatalog) ContainsSanitizer(s string) bool { return containsAny(s, "sanitize", "escape") }

func TestPropagateTaintDirectSourceToSink(t *testing.T) {
	g, err := NormalizeSource("app.js", []byte("const id = req.query.id; db.query(id);"))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	results := PropagateTaint(g, testCatalog{})
	if len(results) != 1 {
		t.Fatalf("expected one taint result, got %d: %+v", len(results), results)
	}
	if results[0].Source.Name != "id" || results[0].Sink.Name != "db.query" {
		t.Fatalf("unexpected taint endpoints: %+v", results[0])
	}
	if results[0].ConfidenceScore <= 0 || results[0].ConfidenceScore > 1 {
		t.Fatalf("confidence out of range: %f", results[0].ConfidenceScore)
	}
}

func TestPropagateTaintThroughThreeAssignments(t *testing.T) {
	source := []byte(`const a = req.query.id;
const b = a;
const c = b;
const d = c;
db.query(d);`)
	g, err := NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	results := PropagateTaint(g, testCatalog{})
	if len(results) != 1 {
		t.Fatalf("expected one taint result, got %d: %+v", len(results), results)
	}
	if len(results[0].Path) < 5 {
		t.Fatalf("expected full source-to-sink path through assignments, got %+v", results[0].Path)
	}
	if results[0].ConfidenceScore >= 0.95 {
		t.Fatalf("transitive confidence should decay below direct source confidence")
	}
}

func TestPropagateTaintClearsSanitizedValue(t *testing.T) {
	g, err := NormalizeSource("app.js", []byte("const id = req.query.id; const safe = sanitize(id); db.query(safe);"))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if results := PropagateTaint(g, testCatalog{}); len(results) != 0 {
		t.Fatalf("sanitized value should not produce findings: %+v", results)
	}
}

func TestPropagateTaintCleanCode(t *testing.T) {
	g, err := NormalizeSource("app.js", []byte("const id = '42'; db.query(id);"))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if results := PropagateTaint(g, testCatalog{}); len(results) != 0 {
		t.Fatalf("clean code should not produce findings: %+v", results)
	}
}

func TestPropagateTaintHandlesNilInputs(t *testing.T) {
	if results := PropagateTaint(nil, testCatalog{}); len(results) != 0 {
		t.Fatalf("nil graph should produce no results")
	}
	g := &Graph{}
	if results := PropagateTaint(g, nil); len(results) != 0 {
		t.Fatalf("nil catalog should produce no results")
	}
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if stringsContains(s, needle) {
			return true
		}
	}
	return false
}

func stringsContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && regexpIndex(s, substr) >= 0)
}

func regexpIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
