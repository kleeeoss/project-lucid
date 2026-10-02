package rules

import (
	"strings"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
)

const (
	CommandInjectionRuleID   = "LUCID-SEC-002"
	CommandInjectionRuleName = "Command Injection"
	CommandInjectionCWE      = "CWE-78"
	CommandInjectionOWASP    = "A03:2021-Injection"
)

func DetectCommandInjection(filePath string, source []byte) ([]models.Vulnerability, error) {
	g, err := graph.NormalizeSource(filePath, source)
	if err != nil {
		return nil, err
	}
	return DetectCommandInjectionInGraph(g), nil
}

func DetectCommandInjectionInGraph(g *graph.Graph) []models.Vulnerability {
	if g == nil {
		return nil
	}
	catalog := NewCatalog()
	results := graph.PropagateTaint(g, catalog)
	vulns := make([]models.Vulnerability, 0, len(results))
	for _, result := range results {
		sinkKind, ok := catalog.SinkKind(result.Sink)
		if !ok || sinkKind != SinkCommand {
			continue
		}
		if isSafeSubprocessCall(result.Sink.CodeSnippet) {
			continue
		}
		vulns = append(vulns, models.Vulnerability{
			ID:              deterministicVulnerabilityID(CommandInjectionRuleID, result.Sink),
			FilePath:        g.FilePath,
			RuleID:          CommandInjectionRuleID,
			RuleName:        CommandInjectionRuleName,
			CWE:             CommandInjectionCWE,
			OWASP:           CommandInjectionOWASP,
			Severity:        models.SeverityCritical,
			ConfidenceScore: result.ConfidenceScore,
			Description:     "Untrusted input flows into an operating system command execution sink.",
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

func isSafeSubprocessCall(code string) bool {
	lower := strings.ToLower(code)
	if strings.Contains(lower, "shell=false") {
		return true
	}
	return false
}
