package parser

import (
	"errors"
	"fmt"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	tsjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

type Language string

const (
	LanguageJavaScript Language = "javascript"
)

type Parser struct {
	inner    *treesitter.Parser
	language Language
	closed   bool
}

type Tree struct {
	inner  *treesitter.Tree
	source []byte
}

type NodeInfo struct {
	Kind        string
	Named       bool
	HasError    bool
	IsError     bool
	StartByte   uint
	EndByte     uint
	StartLine   uint
	StartColumn uint
	EndLine     uint
	EndColumn   uint
	Depth       int
	Snippet     string
}

func NewJavaScriptParser() (*Parser, error) {
	return NewParser(LanguageJavaScript)
}

func NewParser(language Language) (*Parser, error) {
	if language != LanguageJavaScript {
		return nil, fmt.Errorf("unsupported parser language %q", language)
	}
	inner := treesitter.NewParser()
	if inner == nil {
		return nil, errors.New("tree-sitter returned a nil parser")
	}
	if err := inner.SetLanguage(treesitter.NewLanguage(tsjavascript.Language())); err != nil {
		inner.Close()
		return nil, fmt.Errorf("set %s grammar: %w", language, err)
	}
	return &Parser{inner: inner, language: language}, nil
}

func (p *Parser) Close() {
	if p == nil || p.closed {
		return
	}
	if p.inner != nil {
		p.inner.Close()
	}
	p.inner = nil
	p.closed = true
}

func (p *Parser) Parse(source []byte) (*Tree, error) {
	if p == nil || p.inner == nil || p.closed {
		return nil, errors.New("parse with closed or nil parser")
	}
	if source == nil {
		source = []byte{}
	}
	parsed := p.inner.Parse(source, nil)
	if parsed == nil {
		return nil, errors.New("tree-sitter returned a nil tree")
	}
	copySource := append([]byte(nil), source...)
	return &Tree{inner: parsed, source: copySource}, nil
}

func (t *Tree) Close() {
	if t == nil {
		return
	}
	if t.inner != nil {
		t.inner.Close()
	}
	t.inner = nil
	t.source = nil
}

func (t *Tree) Root() (NodeInfo, error) {
	if t == nil || t.inner == nil {
		return NodeInfo{}, errors.New("root requested from nil or closed tree")
	}
	root := t.inner.RootNode()
	if root == nil {
		return NodeInfo{}, errors.New("tree-sitter returned a nil root node")
	}
	return nodeInfo(root, t.source, 0), nil
}

func (t *Tree) HasError() bool {
	if t == nil || t.inner == nil {
		return false
	}
	root := t.inner.RootNode()
	return root != nil && root.HasError()
}

func (t *Tree) WalkNamed(maxDepth int) ([]NodeInfo, error) {
	if t == nil || t.inner == nil {
		return nil, errors.New("walk requested from nil or closed tree")
	}
	root := t.inner.RootNode()
	if root == nil {
		return nil, errors.New("tree-sitter returned a nil root node")
	}
	if maxDepth < 0 {
		maxDepth = 0
	}
	nodes := make([]NodeInfo, 0, 32)
	var visit func(*treesitter.Node, int)
	visit = func(n *treesitter.Node, depth int) {
		if n == nil || depth > maxDepth {
			return
		}
		if n.IsNamed() || depth == 0 || n.IsError() {
			nodes = append(nodes, nodeInfo(n, t.source, depth))
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			visit(n.Child(i), depth+1)
		}
	}
	visit(root, 0)
	return nodes, nil
}

func nodeInfo(n *treesitter.Node, source []byte, depth int) NodeInfo {
	start := n.StartPosition()
	end := n.EndPosition()
	startByte := n.StartByte()
	endByte := n.EndByte()
	return NodeInfo{
		Kind:        n.Kind(),
		Named:       n.IsNamed(),
		HasError:    n.HasError(),
		IsError:     n.IsError(),
		StartByte:   startByte,
		EndByte:     endByte,
		StartLine:   start.Row + 1,
		StartColumn: start.Column + 1,
		EndLine:     end.Row + 1,
		EndColumn:   end.Column + 1,
		Depth:       depth,
		Snippet:     snippet(source, startByte, endByte),
	}
}

func snippet(source []byte, start, end uint) string {
	if start > end || int(start) > len(source) || int(end) > len(source) {
		return ""
	}
	text := strings.TrimSpace(string(source[start:end]))
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	if len(text) > 80 {
		return text[:77] + "..."
	}
	return text
}
