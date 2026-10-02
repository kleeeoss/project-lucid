package graph

import (
	"regexp"
	"sort"
	"strings"
)

type SecurityCatalog interface {
	IsSource(Node) bool
	IsSink(Node) bool
	IsSanitizer(Node) bool
	ContainsSource(string) bool
	ContainsSink(string) bool
	ContainsSanitizer(string) bool
}

type TaintResult struct {
	Source          Node
	Sink            Node
	Path            []Node
	ConfidenceScore float64
}

type taintState struct {
	source     Node
	path       []Node
	confidence float64
}

func PropagateTaint(g *Graph, catalog SecurityCatalog) []TaintResult {
	if g == nil || catalog == nil || len(g.Nodes) == 0 {
		return nil
	}
	nodes := append([]Node(nil), g.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].ByteStart == nodes[j].ByteStart {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].ByteStart < nodes[j].ByteStart
	})

	tainted := map[string]taintState{}
	results := make([]TaintResult, 0)
	seen := map[string]bool{}

	for _, node := range nodes {
		if isAssignmentLike(node) {
			target := node.Name
			if target == "" {
				continue
			}
			if isSanitized(node, catalog) {
				delete(tainted, target)
				continue
			}
			if isSourceLike(node, catalog) {
				tainted[target] = taintState{source: node, path: []Node{node}, confidence: 0.95}
				continue
			}
			if prior, ok := firstReferencedTaint(node.CodeSnippet, tainted, true); ok {
				tainted[target] = taintState{source: prior.source, path: appendPath(prior.path, node), confidence: decay(prior.confidence)}
				continue
			}
			delete(tainted, target)
		}

		if isSinkLike(node, catalog) && !isSanitized(node, catalog) {
			if isSourceLike(node, catalog) {
				key := node.ID + ":direct"
				if !seen[key] {
					seen[key] = true
					results = append(results, TaintResult{Source: node, Sink: node, Path: []Node{node}, ConfidenceScore: 0.95})
				}
				continue
			}
			if prior, ok := firstReferencedTaint(node.CodeSnippet, tainted, false); ok {
				key := prior.source.ID + "->" + node.ID
				if seen[key] {
					continue
				}
				seen[key] = true
				results = append(results, TaintResult{Source: prior.source, Sink: node, Path: appendPath(prior.path, node), ConfidenceScore: clamp(prior.confidence)})
			}
		}
	}
	return results
}

func isAssignmentLike(n Node) bool {
	return n.Type == SemanticVariableDeclaration || n.Type == SemanticAssignmentExpression
}

func isSourceLike(n Node, catalog SecurityCatalog) bool {
	return catalog.IsSource(n) || catalog.ContainsSource(n.Name) || catalog.ContainsSource(n.CodeSnippet)
}

func isSinkLike(n Node, catalog SecurityCatalog) bool {
	if n.Type != SemanticCallExpression {
		return false
	}
	return catalog.IsSink(n) || catalog.ContainsSink(n.Name) || catalog.ContainsSink(n.CodeSnippet)
}

func isSanitized(n Node, catalog SecurityCatalog) bool {
	return catalog.IsSanitizer(n) || catalog.ContainsSanitizer(n.Name) || catalog.ContainsSanitizer(n.CodeSnippet)
}

func firstReferencedTaint(code string, tainted map[string]taintState, assignment bool) (taintState, bool) {
	if assignment {
		code = expressionPortion(code)
	}
	code = quotedStringRE.ReplaceAllString(code, "")
	vars := identifierRE.FindAllString(code, -1)
	for _, v := range vars {
		if state, ok := tainted[v]; ok {
			return state, true
		}
	}
	return taintState{}, false
}

func expressionPortion(code string) string {
	if idx := strings.Index(code, "="); idx >= 0 && idx+1 < len(code) {
		return code[idx+1:]
	}
	return code
}

func appendPath(path []Node, node Node) []Node {
	out := append([]Node(nil), path...)
	if len(out) == 0 || out[len(out)-1].ID != node.ID {
		out = append(out, node)
	}
	return out
}

func decay(confidence float64) float64 {
	return clamp(confidence - 0.05)
}

func clamp(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

var quotedStringRE = regexp.MustCompile("\"(?:\\\\.|[^\"\\\\])*\"|'(?:\\\\.|[^'\\\\])*'|`(?:\\\\.|[^`\\\\])*`")
