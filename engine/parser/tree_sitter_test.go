package parser

import "testing"

func TestJavaScriptParserParsesValidSource(t *testing.T) {
	p, err := NewJavaScriptParser()
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}
	defer p.Close()

	tree, err := p.Parse([]byte("function handler(req) { const id = req.query.id; return id; }"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Close()

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	if root.Kind != "program" {
		t.Fatalf("root kind = %q, want program", root.Kind)
	}
	if root.HasError {
		t.Fatalf("valid source should not contain errors: %+v", root)
	}
}

func TestJavaScriptParserHandlesMalformedSource(t *testing.T) {
	p, err := NewJavaScriptParser()
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}
	defer p.Close()

	tree, err := p.Parse([]byte("function broken( { const x = ;"))
	if err != nil {
		t.Fatalf("malformed input should still return a tree: %v", err)
	}
	defer tree.Close()

	if !tree.HasError() {
		t.Fatalf("malformed input should be represented by ERROR nodes")
	}
	nodes, err := tree.WalkNamed(4)
	if err != nil {
		t.Fatalf("walk malformed tree: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatalf("expected malformed tree to contain at least the root node")
	}
}

func TestParserLifecycleRejectsUseAfterClose(t *testing.T) {
	p, err := NewJavaScriptParser()
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}
	p.Close()
	p.Close()
	if _, err := p.Parse([]byte("const x = 1;")); err == nil {
		t.Fatalf("expected parse with closed parser to fail")
	}
}

func TestTreeLifecycleRejectsUseAfterClose(t *testing.T) {
	p, err := NewJavaScriptParser()
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}
	defer p.Close()
	tree, err := p.Parse(nil)
	if err != nil {
		t.Fatalf("parse empty source: %v", err)
	}
	tree.Close()
	tree.Close()
	if _, err := tree.Root(); err == nil {
		t.Fatalf("expected root from closed tree to fail")
	}
}

func TestUnsupportedLanguageFails(t *testing.T) {
	if _, err := NewParser(Language("python")); err == nil {
		t.Fatalf("phase 1 parser should reject unsupported language until registry is implemented")
	}
}
