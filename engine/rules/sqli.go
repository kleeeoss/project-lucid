package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
)

const (
	SQLInjectionRuleID   = "LUCID-SEC-001"
	SQLInjectionRuleName = "SQL Injection"
	SQLInjectionCWE      = "CWE-89"
	SQLInjectionOWASP    = "A03:2021-Injection"
)

func DetectSQLInjection(filePath string, source []byte) ([]models.Vulnerability, error) {
	g, err := graph.NormalizeSource(filePath, source)
	if err != nil {
		return nil, err
	}
	return DetectSQLInjectionInGraph(g), nil
}

func DetectSQLInjectionInGraph(g *graph.Graph) []models.Vulnerability {
	if g == nil {
		return nil
	}
	catalog := NewCatalog()
	results := graph.PropagateTaint(g, catalog)
	vulns := make([]models.Vulnerability, 0, len(results))
	for _, result := range results {
		sinkKind, ok := catalog.SinkKind(result.Sink)
		if !ok || sinkKind != SinkSQL {
			continue
		}
		if isParameterizedSQL(result.Sink.CodeSnippet) || !hasUnsafeSQLConstruction(result) {
			continue
		}
		vulns = append(vulns, models.Vulnerability{
			ID:              deterministicVulnerabilityID(SQLInjectionRuleID, result.Sink),
			FilePath:        g.FilePath,
			RuleID:          SQLInjectionRuleID,
			RuleName:        SQLInjectionRuleName,
			CWE:             SQLInjectionCWE,
			OWASP:           SQLInjectionOWASP,
			Severity:        models.SeverityHigh,
			ConfidenceScore: result.ConfidenceScore,
			Description:     "Untrusted input flows into a raw SQL query without parameterization.",
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

func hasUnsafeSQLConstruction(result graph.TaintResult) bool {
	for _, node := range result.Path {
		code := strings.ToLower(node.CodeSnippet)
		if strings.Contains(code, "select") || strings.Contains(code, "insert") || strings.Contains(code, "update") || strings.Contains(code, "delete") {
			if strings.Contains(node.CodeSnippet, "+") || strings.Contains(node.CodeSnippet, "${") || strings.Contains(code, "f\"") || strings.Contains(code, "f'") {
				return true
			}
		}
	}
	code := strings.ToLower(result.Sink.CodeSnippet)
	return (strings.Contains(code, "select") || strings.Contains(code, "insert") || strings.Contains(code, "update") || strings.Contains(code, "delete")) &&
		(strings.Contains(result.Sink.CodeSnippet, "+") || strings.Contains(result.Sink.CodeSnippet, "${") || strings.Contains(code, "f\"") || strings.Contains(code, "f'"))
}

func isParameterizedSQL(code string) bool {
	lower := strings.ToLower(code)
	if strings.Contains(lower, "execute(") && strings.Contains(code, ",") && !strings.Contains(code, "+") && !strings.Contains(code, "${") {
		return true
	}
	if strings.Contains(lower, "query(") && strings.Contains(code, ",") && (strings.Contains(code, "?") || strings.Contains(code, "$1")) && !strings.Contains(code, "+") && !strings.Contains(code, "${") {
		return true
	}
	return false
}

func deterministicVulnerabilityID(ruleID string, sink graph.Node) string {
	return fmt.Sprintf("%s:%s:%d:%d", ruleID, filepath.ToSlash(sink.FilePath), sink.ByteStart, sink.ByteEnd)
}

func astRef(node graph.Node) models.ASTNodeRef {
	return models.ASTNodeRef{
		Type:      string(node.Type),
		Name:      node.Name,
		Line:      node.LineStart,
		Column:    node.ColumnStart,
		ByteStart: node.ByteStart,
		ByteEnd:   node.ByteEnd,
	}
}

func astPath(path []graph.Node) []models.ASTNodeRef {
	out := make([]models.ASTNodeRef, 0, len(path))
	for _, node := range path {
		out = append(out, astRef(node))
	}
	return out
}
