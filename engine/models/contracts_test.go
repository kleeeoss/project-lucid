package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestVulnerabilityReportJSONContract(t *testing.T) {
	report := VulnerabilityReport{
		ScanID:         "scan-001",
		FilePath:       "app.js",
		Language:       "javascript",
		ScanDurationMs: 12,
		Timestamp:      time.Date(2026, time.September, 20, 1, 2, 3, 0, time.UTC),
		Vulnerabilities: []Vulnerability{{
			ID:              "vuln-001",
			FilePath:        "app.js",
			RuleID:          "LUCID-SEC-001",
			RuleName:        "SQL Injection",
			CWE:             "CWE-89",
			OWASP:           "A03:2021-Injection",
			Severity:        SeverityHigh,
			ConfidenceScore: 0.9,
			Description:     "description",
			SourceNode:      ASTNodeRef{Type: "Identifier", Name: "id", Line: 1, Column: 2, ByteStart: 3, ByteEnd: 5},
			SinkNode:        ASTNodeRef{Type: "CallExpression", Name: "db.query", Line: 2, Column: 4, ByteStart: 9, ByteEnd: 17},
			TaintPath:       []ASTNodeRef{{Type: "Identifier", Name: "id", Line: 1, Column: 2, ByteStart: 3, ByteEnd: 5}},
			VulnerableCode:  "db.query(q)",
			LineStart:       2,
			LineEnd:         2,
			ColStart:        4,
			ColEnd:          15,
		}},
	}

	b, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	jsonText := string(b)
	for _, required := range []string{"\"scan_id\"", "\"file_path\"", "\"vulnerabilities\"", "\"confidence_score\"", "\"byte_start\""} {
		if !strings.Contains(jsonText, required) {
			t.Fatalf("expected JSON to contain %s: %s", required, jsonText)
		}
	}
	if strings.Contains(jsonText, "target_file") {
		t.Fatalf("target_file should be omitted when empty: %s", jsonText)
	}
}

func TestSeverityConstants(t *testing.T) {
	cases := map[SeverityLevel]string{
		SeverityCritical: "CRITICAL",
		SeverityHigh:     "HIGH",
		SeverityMedium:   "MEDIUM",
		SeverityLow:      "LOW",
		SeverityInfo:     "INFO",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Fatalf("severity mismatch: got %q want %q", got, want)
		}
	}
}
