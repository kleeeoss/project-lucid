package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestEngineAnalyzer_AnalyzeFile_SQLi(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	analyzer := NewEngineAnalyzer(logger)

	// Vulnerable code snippet from Apurv's test fixtures
	vulnerableJS := []byte(`function getUser(req) {
  const userId = req.query.id;
  const sql = "SELECT * FROM users WHERE id = " + userId;
  db.query(sql);
}`)

	vulns, err := analyzer.AnalyzeFile(context.Background(), "controllers/user.js", vulnerableJS)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	if len(vulns) == 0 {
		t.Fatalf("expected at least 1 vulnerability detected by Apurv's engine, got 0")
	}

	v := vulns[0]
	if v.RuleID != "LUCID-SEC-001" || v.CWE != "CWE-89" {
		t.Errorf("unexpected finding metadata: RuleID=%s, CWE=%s", v.RuleID, v.CWE)
	}

	taintSummary := FormatTaintPath(v.TaintPath)
	if len(taintSummary) == 0 {
		t.Errorf("expected non-empty taint path summary")
	}
}

func TestEngineAnalyzer_AnalyzeFile_Clean(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	analyzer := NewEngineAnalyzer(logger)

	// Safe parameterized query
	cleanJS := []byte(`function getUser(req) {
  const userId = req.query.id;
  db.query("SELECT * FROM users WHERE id = ?", [userId]);
}`)

	vulns, err := analyzer.AnalyzeFile(context.Background(), "controllers/user.js", cleanJS)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	if len(vulns) != 0 {
		t.Errorf("expected 0 vulnerabilities in clean code, got %d", len(vulns))
	}
}
