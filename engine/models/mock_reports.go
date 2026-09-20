package models

import "time"

var mockTimestamp = time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)

func MockSQLInjectionReport() VulnerabilityReport {
	filePath := "fixtures/vulnerable_sqli.js"
	source := ASTNodeRef{Type: "MemberExpression", Name: "req.query.id", Line: 2, Column: 14, ByteStart: 42, ByteEnd: 54}
	sink := ASTNodeRef{Type: "CallExpression", Name: "db.query", Line: 4, Column: 3, ByteStart: 98, ByteEnd: 106}
	query := ASTNodeRef{Type: "VariableDeclaration", Name: "query", Line: 3, Column: 9, ByteStart: 61, ByteEnd: 96}

	return VulnerabilityReport{
		ScanID:         "scan-mock-sqli-001",
		FilePath:       filePath,
		TargetFile:     filePath,
		Language:       "javascript",
		ScanDurationMs: 7,
		Timestamp:      mockTimestamp,
		Vulnerabilities: []Vulnerability{
			{
				ID:              "vuln-mock-sqli-001",
				FilePath:        filePath,
				RuleID:          "LUCID-SEC-001",
				RuleName:        "SQL Injection",
				CWE:             "CWE-89",
				OWASP:           "A03:2021-Injection",
				Severity:        SeverityHigh,
				ConfidenceScore: 0.95,
				Description:     "Untrusted request data flows into a raw SQL query without parameterization.",
				SourceNode:      source,
				SinkNode:        sink,
				TaintPath:       []ASTNodeRef{source, query, sink},
				VulnerableCode:  "db.query(query)",
				LineStart:       4,
				LineEnd:         4,
				ColStart:        3,
				ColEnd:          18,
			},
		},
	}
}

func MockCommandInjectionReport() VulnerabilityReport {
	filePath := "fixtures/vulnerable_cmd.py"
	source := ASTNodeRef{Type: "CallExpression", Name: "request.args.get", Line: 5, Column: 10, ByteStart: 104, ByteEnd: 127}
	sink := ASTNodeRef{Type: "CallExpression", Name: "os.system", Line: 6, Column: 4, ByteStart: 132, ByteEnd: 141}

	return VulnerabilityReport{
		ScanID:         "scan-mock-cmd-001",
		FilePath:       filePath,
		TargetFile:     filePath,
		Language:       "python",
		ScanDurationMs: 8,
		Timestamp:      mockTimestamp,
		Vulnerabilities: []Vulnerability{
			{
				ID:              "vuln-mock-cmd-001",
				FilePath:        filePath,
				RuleID:          "LUCID-SEC-002",
				RuleName:        "Command Injection",
				CWE:             "CWE-78",
				OWASP:           "A03:2021-Injection",
				Severity:        SeverityCritical,
				ConfidenceScore: 0.98,
				Description:     "Untrusted request data reaches an operating system command execution sink.",
				SourceNode:      source,
				SinkNode:        sink,
				TaintPath:       []ASTNodeRef{source, sink},
				VulnerableCode:  "os.system(cmd)",
				LineStart:       6,
				LineEnd:         6,
				ColStart:        4,
				ColEnd:          18,
			},
		},
	}
}
