package rules

import (
	"strings"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
)

const (
	PathTraversalRuleID   = "LUCID-SEC-004"
	PathTraversalRuleName = "Path Traversal"
	PathTraversalCWE      = "CWE-22"
	PathTraversalOWASP    = "A01:2021-Broken Access Control"
)

func DetectPathTraversal(filePath string, source []byte) ([]models.Vulnerability, error) {
	g, err := graph.NormalizeSource(filePath, source)
	if err != nil {
		return nil, err
	}
	return DetectPathTraversalInGraph(g), nil
}

func DetectPathTraversalInGraph(g *graph.Graph) []models.Vulnerability {
	if g == nil {
		return nil
	}
	catalog := NewCatalog()
	results := graph.PropagateTaint(g, catalog)
	vulns := make([]models.Vulnerability, 0, len(results))
	for _, result := range results {
		sinkKind, ok := catalog.SinkKind(result.Sink)
		if !ok || sinkKind != SinkFile {
			continue
		}
		if hasBoundaryCheck(result.Path) || hasBoundaryCheck([]graph.Node{result.Sink}) {
			continue
		}
		vulns = append(vulns, models.Vulnerability{
			ID:              deterministicVulnerabilityID(PathTraversalRuleID, result.Sink),
			FilePath:        g.FilePath,
			RuleID:          PathTraversalRuleID,
			RuleName:        PathTraversalRuleName,
			CWE:             PathTraversalCWE,
			OWASP:           PathTraversalOWASP,
			Severity:        models.SeverityHigh,
			ConfidenceScore: result.ConfidenceScore,
			Description:     "Untrusted path input reaches a filesystem sink without boundary verification.",
			SourceNode:      astRef(result.Source),
			SinkNode:        astRef(result.Sink),
			TaintPath:       astPath(result.Path),
			VulnerableCode:  result.Sink.CodeSnippet,
			LineStart:       result.Sink.LineStart,
			LineEnd:         result.Sink.LineEnd,
			ColStart:        result.Sink.ColumnStart,
			ColEnd:          result.Sink.ColumnEnd,
		})
	}
	return vulns
}

func hasBoundaryCheck(path []graph.Node) bool {
	for _, node := range path {
		code := strings.ToLower(node.CodeSnippet)
		if strings.Contains(code, "path.resolve") || strings.Contains(code, "path.normalize") || strings.Contains(code, "os.path.abspath") || strings.Contains(code, "resolve()") {
			return true
		}
		if strings.Contains(code, "startswith(") || strings.Contains(code, "startsWith(") {
			return true
		}
	}
	return false
}
