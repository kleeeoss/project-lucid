package graph

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"lucid-ci/engine/parser"
)

type SemanticType string

const (
	SemanticProgram              SemanticType = "Program"
	SemanticFunctionDeclaration  SemanticType = "FunctionDeclaration"
	SemanticVariableDeclaration  SemanticType = "VariableDeclaration"
	SemanticAssignmentExpression SemanticType = "AssignmentExpression"
	SemanticCallExpression       SemanticType = "CallExpression"
	SemanticIdentifier           SemanticType = "Identifier"
	SemanticError                SemanticType = "Error"
)

type Node struct {
	ID            string       `json:"id"`
	Type          SemanticType `json:"type"`
	CSTKind       string       `json:"cst_kind"`
	Name          string       `json:"name"`
	FilePath      string       `json:"file_path"`
	ByteStart     uint32       `json:"byte_start"`
	ByteEnd       uint32       `json:"byte_end"`
	LineStart     int          `json:"line_start"`
	ColumnStart   int          `json:"column_start"`
	LineEnd       int          `json:"line_end"`
	ColumnEnd     int          `json:"column_end"`
	CodeSnippet   string       `json:"code_snippet"`
	ParentID      string       `json:"parent_id,omitempty"`
	ChildIDs      []string     `json:"child_ids,omitempty"`
	ContainsError bool         `json:"contains_error"`
	Depth         int          `json:"-"`
}

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type Graph struct {
	FilePath string `json:"file_path"`
	Language string `json:"language"`
	Nodes    []Node `json:"nodes"`
	Edges    []Edge `json:"edges"`
}

func NormalizeSource(filePath string, source []byte) (*Graph, error) {
	filePath = filepath.ToSlash(strings.TrimSpace(filePath))
	if filePath == "" {
		return nil, fmt.Errorf("normalize source: empty file path")
	}
	p, spec, err := parser.NewParserForPath(filePath)
	if err != nil {
		return nil, err
	}
	defer p.Close()
	tree, err := p.Parse(source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	cstNodes, err := tree.WalkNamed(512)
	if err != nil {
		return nil, err
	}
	g := &Graph{FilePath: filePath, Language: spec.Name, Nodes: make([]Node, 0, len(cstNodes)), Edges: make([]Edge, 0, len(cstNodes))}
	parentStack := make([]int, 0, 32)
	for _, info := range cstNodes {
		semantic, ok := semanticType(spec.Language, info.Kind)
		if !ok {
			continue
		}
		for len(parentStack) > 0 && info.Depth <= g.Nodes[parentStack[len(parentStack)-1]].Depth {
			parentStack = parentStack[:len(parentStack)-1]
		}

		idx := len(g.Nodes)
		node := Node{
			ID:            nodeID(filePath, semantic, info, idx),
			Type:          semantic,
			CSTKind:       info.Kind,
			Name:          inferName(semantic, info.Snippet),
			FilePath:      filePath,
			ByteStart:     uint32(info.StartByte),
			ByteEnd:       uint32(info.EndByte),
			LineStart:     int(info.StartLine),
			ColumnStart:   int(info.StartColumn),
			LineEnd:       int(info.EndLine),
			ColumnEnd:     int(info.EndColumn),
			CodeSnippet:   info.Snippet,
			ContainsError: info.HasError || info.IsError,
			Depth:         info.Depth,
		}
		if len(parentStack) > 0 {
			parentIdx := parentStack[len(parentStack)-1]
			node.ParentID = g.Nodes[parentIdx].ID
			g.Nodes[parentIdx].ChildIDs = append(g.Nodes[parentIdx].ChildIDs, node.ID)
			g.Edges = append(g.Edges, Edge{ID: fmt.Sprintf("edge:%s->%s", g.Nodes[parentIdx].ID, node.ID), Source: g.Nodes[parentIdx].ID, Target: node.ID, Kind: "ast_child"})
		}
		g.Nodes = append(g.Nodes, node)
		parentStack = append(parentStack, idx)
	}
	return g, nil
}

func semanticType(language parser.Language, kind string) (SemanticType, bool) {
	if kind == "ERROR" || kind == "ERROR_SENTINEL" {
		return SemanticError, true
	}
	switch language {
	case parser.LanguageJavaScript:
		switch kind {
		case "program":
			return SemanticProgram, true
		case "function_declaration", "arrow_function", "method_definition":
			return SemanticFunctionDeclaration, true
		case "lexical_declaration", "variable_declaration", "variable_declarator":
			return SemanticVariableDeclaration, true
		case "assignment_expression", "augmented_assignment_expression":
			return SemanticAssignmentExpression, true
		case "call_expression":
			return SemanticCallExpression, true
		case "identifier", "property_identifier", "shorthand_property_identifier":
			return SemanticIdentifier, true
		}
	case parser.LanguagePython:
		switch kind {
		case "module":
			return SemanticProgram, true
		case "function_definition":
			return SemanticFunctionDeclaration, true
		case "assignment":
			return SemanticAssignmentExpression, true
		case "call":
			return SemanticCallExpression, true
		case "identifier":
			return SemanticIdentifier, true
		}
	}
	return "", false
}

func nodeID(filePath string, semantic SemanticType, info parser.NodeInfo, index int) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", filePath, semantic, info.StartByte, info.EndByte, info.StartLine, index)
}

var identifierRE = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

func inferName(semantic SemanticType, snippet string) string {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return ""
	}
	if semantic == SemanticIdentifier {
		return snippet
	}
	if semantic == SemanticFunctionDeclaration {
		if strings.HasPrefix(snippet, "def ") {
			return firstIdentifier(strings.TrimPrefix(snippet, "def "))
		}
		if strings.HasPrefix(snippet, "function ") {
			return firstIdentifier(strings.TrimPrefix(snippet, "function "))
		}
	}
	if semantic == SemanticVariableDeclaration || semantic == SemanticAssignmentExpression {
		return firstIdentifier(snippet)
	}
	if semantic == SemanticCallExpression {
		if idx := strings.Index(snippet, "("); idx > 0 {
			return strings.TrimSpace(snippet[:idx])
		}
	}
	return firstIdentifier(snippet)
}

func firstIdentifier(text string) string {
	match := identifierRE.FindString(text)
	return match
}
