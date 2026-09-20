package graph

import "testing"

func TestNormalizeJavaScriptSemanticGraph(t *testing.T) {
	source := []byte(`function handler(req) {
  const id = req.query.id;
  const query = "SELECT " + id;
  db.query(query);
}`)
	g, err := NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("NormalizeSource JS: %v", err)
	}
	assertGraphHas(t, g, SemanticFunctionDeclaration, "handler")
	assertGraphHas(t, g, SemanticVariableDeclaration, "id")
	assertGraphHas(t, g, SemanticCallExpression, "db.query")
	if g.Language != "javascript" || g.FilePath != "app.js" {
		t.Fatalf("unexpected graph metadata: %+v", g)
	}
	if len(g.Nodes) == 0 || len(g.Edges) == 0 {
		t.Fatalf("expected normalized nodes and edges, got nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
	}
	for _, n := range g.Nodes {
		if n.ID == "" || n.ByteEnd < n.ByteStart || n.LineStart <= 0 || n.ColumnStart <= 0 {
			t.Fatalf("invalid node positions: %+v", n)
		}
		if n.CSTKind == "{" || n.CSTKind == "}" || n.CSTKind == ";" {
			t.Fatalf("anonymous syntax token leaked into semantic graph: %+v", n)
		}
	}
}

func TestNormalizePythonSemanticGraph(t *testing.T) {
	source := []byte("def handler(request):\n    name = request.args.get('name')\n    cursor.execute('SELECT ' + name)\n")
	g, err := NormalizeSource("views.py", source)
	if err != nil {
		t.Fatalf("NormalizeSource Python: %v", err)
	}
	assertGraphHas(t, g, SemanticFunctionDeclaration, "handler")
	assertGraphHas(t, g, SemanticAssignmentExpression, "name")
	assertGraphHas(t, g, SemanticCallExpression, "cursor.execute")
	if g.Language != "python" {
		t.Fatalf("unexpected language: %s", g.Language)
	}
}

func TestNormalizeMalformedSourceDoesNotPanic(t *testing.T) {
	g, err := NormalizeSource("broken.js", []byte("function broken( { const x = ;"))
	if err != nil {
		t.Fatalf("NormalizeSource malformed JS should produce a graph: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatalf("malformed source should still produce root/error graph nodes")
	}
	foundError := false
	for _, n := range g.Nodes {
		if n.ContainsError {
			foundError = true
		}
	}
	if !foundError {
		t.Fatalf("malformed graph should mark error-containing nodes")
	}
}

func TestNormalizeIsDeterministic(t *testing.T) {
	source := []byte("const x = req.query.id; db.query(x);")
	g1, err := NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("first normalize: %v", err)
	}
	g2, err := NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("second normalize: %v", err)
	}
	if len(g1.Nodes) != len(g2.Nodes) || len(g1.Edges) != len(g2.Edges) {
		t.Fatalf("non-deterministic graph sizes")
	}
	for i := range g1.Nodes {
		if g1.Nodes[i].ID != g2.Nodes[i].ID || g1.Nodes[i].Name != g2.Nodes[i].Name {
			t.Fatalf("node %d not deterministic: %+v vs %+v", i, g1.Nodes[i], g2.Nodes[i])
		}
	}
}

func TestNormalizeRejectsUnsupportedFile(t *testing.T) {
	if _, err := NormalizeSource("app.rb", []byte("puts 'x'")); err == nil {
		t.Fatalf("expected unsupported extension to fail")
	}
}

func assertGraphHas(t *testing.T, g *Graph, typ SemanticType, name string) {
	t.Helper()
	for _, n := range g.Nodes {
		if n.Type == typ && n.Name == name {
			return
		}
	}
	t.Fatalf("expected graph to contain %s named %q; nodes=%+v", typ, name, g.Nodes)
}
