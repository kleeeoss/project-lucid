package worker

import (
	"context"
	"fmt"
	"log/slog"

	engineModels "lucid-ci/engine/models"
	"lucid-ci/engine/rules"
	"lucid-ci/platform/models"
)

// EngineAnalyzer defines the interface for running Apurv's AST security rules.
type EngineAnalyzer interface {
	AnalyzeFile(ctx context.Context, filePath string, content []byte) ([]engineModels.Vulnerability, error)
}

type engineAnalyzer struct {
	logger *slog.Logger
}

// NewEngineAnalyzer initializes the analyzer that calls Apurv's engine in-process.
func NewEngineAnalyzer(logger *slog.Logger) EngineAnalyzer {
	return &engineAnalyzer{logger: logger}
}

// AnalyzeFile executes all built-in OWASP rules on the provided code content.
func (a *engineAnalyzer) AnalyzeFile(ctx context.Context, filePath string, content []byte) ([]engineModels.Vulnerability, error) {
	var allVulns []engineModels.Vulnerability

	// 1. Rule: SQL Injection (LUCID-SEC-001)
	if sqliVulns, err := rules.DetectSQLInjection(filePath, content); err == nil {
		allVulns = append(allVulns, sqliVulns...)
	} else {
		a.logger.Debug("SQLi detector skipped/error", "file", filePath, "error", err)
	}

	// 2. Rule: Command Injection (LUCID-SEC-002)
	if cmdVulns, err := rules.DetectCommandInjection(filePath, content); err == nil {
		allVulns = append(allVulns, cmdVulns...)
	} else {
		a.logger.Debug("Command Injection detector skipped/error", "file", filePath, "error", err)
	}

	// 3. Rule: Insecure Secrets (LUCID-SEC-003)
	if secretVulns, err := rules.DetectInsecureSecrets(filePath, content); err == nil {
		allVulns = append(allVulns, secretVulns...)
	} else {
		a.logger.Debug("Insecure Secrets detector skipped/error", "file", filePath, "error", err)
	}

	// 4. Rule: Path Traversal (LUCID-SEC-004)
	if pathVulns, err := rules.DetectPathTraversal(filePath, content); err == nil {
		allVulns = append(allVulns, pathVulns...)
	} else {
		a.logger.Debug("Path Traversal detector skipped/error", "file", filePath, "error", err)
	}

	a.logger.Info("AST Engine analysis completed",
		"file", filePath,
		"findings_count", len(allVulns),
	)

	return allVulns, nil
}

// FormatTaintPath converts Apurv's ASTNodeRef chain into human-readable strings for Garv's AI prompt.
func FormatTaintPath(nodes []engineModels.ASTNodeRef) []string {
	if len(nodes) == 0 {
		return nil
	}
	summary := make([]string, 0, len(nodes))
	for _, n := range nodes {
		summary = append(summary, fmt.Sprintf("%s (%s at Line %d)", n.Name, n.Type, n.Line))
	}
	return summary
}

// ConvertToPlatformVuln maps Apurv's engine vulnerability to Krish's database model.
func ConvertToPlatformVuln(scanRunID string, v engineModels.Vulnerability) models.Vulnerability {
	return models.Vulnerability{
		ScanRunID:       scanRunID,
		RuleID:          v.RuleID,
		CWE:             v.CWE,
		Severity:        models.VulnSeverity(v.Severity),
		ConfidenceScore: v.ConfidenceScore,
		FilePath:        v.FilePath,
		LineStart:       v.LineStart,
		LineEnd:         v.LineEnd,
		VulnerableCode:  v.VulnerableCode,
	}
}
