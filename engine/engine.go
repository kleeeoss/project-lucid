package engine

import (
	"fmt"
	"time"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
	"lucid-ci/engine/rules"
)

// Analyze performs end-to-end AST security analysis on the provided source file.
// It parses the file, normalizes the AST into a semantic graph, runs intraprocedural
// taint analysis, and evaluates all OWASP vulnerability detectors:
// - SQL Injection (LUCID-SEC-001)
// - Command Injection (LUCID-SEC-002)
// - Insecure Secrets (LUCID-SEC-003)
// - Path Traversal (LUCID-SEC-004)
func Analyze(filePath string, source []byte) ([]models.Vulnerability, error) {
	if len(source) == 0 {
		return nil, nil
	}
	g, err := graph.NormalizeSource(filePath, source)
	if err != nil {
		return nil, fmt.Errorf("engine analyze %q: %w", filePath, err)
	}
	return AnalyzeGraph(g), nil
}

// AnalyzeGraph evaluates all security rules against a pre-normalized AST graph.
func AnalyzeGraph(g *graph.Graph) []models.Vulnerability {
	if g == nil {
		return nil
	}
	var findings []models.Vulnerability
	findings = append(findings, rules.DetectSQLInjectionInGraph(g)...)
	findings = append(findings, rules.DetectCommandInjectionInGraph(g)...)
	findings = append(findings, rules.DetectInsecureSecretsInGraph(g)...)
	findings = append(findings, rules.DetectPathTraversalInGraph(g)...)
	return findings
}

// AnalyzeReport generates a full VulnerabilityReport with scan metadata and timing.
func AnalyzeReport(scanID string, filePath string, source []byte) (*models.VulnerabilityReport, error) {
	start := time.Now()
	vulns, err := Analyze(filePath, source)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	return &models.VulnerabilityReport{
		ScanID:          scanID,
		FilePath:        filePath,
		Vulnerabilities: vulns,
		ScanDurationMs:  duration,
		Timestamp:       time.Now().UTC(),
	}, nil
}
