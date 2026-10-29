package engine

import (
	"testing"

	"lucid-ci/engine/rules"
)

func TestAnalyze_EmptySource(t *testing.T) {
	vulns, err := Analyze("app.js", []byte(""))
	if err != nil {
		t.Fatalf("unexpected error for empty source: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("expected 0 vulnerabilities for empty source, got %d", len(vulns))
	}
}

func TestAnalyze_UnsupportedExtension(t *testing.T) {
	_, err := Analyze("unsupported.rb", []byte("puts 'hello'"))
	if err == nil {
		t.Fatalf("expected error for unsupported file extension")
	}
}

func TestAnalyze_JavaScriptVulnerabilities(t *testing.T) {
	source := []byte(`
const id = req.query.id;
const q = "SELECT * FROM users WHERE id = " + id;
db.query(q);

const host = req.query.host;
const cmd = "ping " + host;
child_process.exec(cmd);

const token = "AKIA1234567890ABCDEF";

const file = req.query.file;
fs.readFile(file);
`)
	vulns, err := Analyze("server.js", source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(vulns) < 4 {
		t.Fatalf("expected at least 4 vulnerabilities (SQLi, Cmd, Secret, Path), got %d: %+v", len(vulns), vulns)
	}

	foundRules := map[string]bool{}
	for _, v := range vulns {
		foundRules[v.RuleID] = true
	}
	expected := []string{
		rules.SQLInjectionRuleID,
		rules.CommandInjectionRuleID,
		rules.InsecureSecretsRuleID,
		rules.PathTraversalRuleID,
	}
	for _, ruleID := range expected {
		if !foundRules[ruleID] {
			t.Errorf("missing expected rule finding: %s", ruleID)
		}
	}
}

func TestAnalyze_PythonVulnerabilities(t *testing.T) {
	source := []byte(`
name = request.args.get('name')
q = 'SELECT * FROM users WHERE name = ' + name
cursor.execute(q)

target = request.args.get('target')
cmd = 'ping ' + target
os.system(cmd)

api_key = "ghp_123456789012345678901234567890123456"

filename = request.args.get('path')
open(filename)
`)
	vulns, err := Analyze("views.py", source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(vulns) < 4 {
		t.Fatalf("expected at least 4 vulnerabilities in Python source, got %d: %+v", len(vulns), vulns)
	}

	foundRules := map[string]bool{}
	for _, v := range vulns {
		foundRules[v.RuleID] = true
	}
	expected := []string{
		rules.SQLInjectionRuleID,
		rules.CommandInjectionRuleID,
		rules.InsecureSecretsRuleID,
		rules.PathTraversalRuleID,
	}
	for _, ruleID := range expected {
		if !foundRules[ruleID] {
			t.Errorf("missing expected rule finding in Python: %s", ruleID)
		}
	}
}

func TestAnalyze_CleanCode(t *testing.T) {
	source := []byte(`
const express = require('express');
const app = express();
app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});
db.query("SELECT * FROM users");
`)
	vulns, err := Analyze("clean.js", source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("expected 0 vulnerabilities in clean code, got %d: %+v", len(vulns), vulns)
	}
}

func TestAnalyze_MalformedCodeNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Analyze panicked on malformed source: %v", r)
		}
	}()

	broken := []byte("function broken( { def [ { x = + ; return )")
	vulns, err := Analyze("broken.js", broken)
	if err != nil {
		t.Fatalf("unexpected hard error on malformed source (tree-sitter should recover): %v", err)
	}
	t.Logf("malformed analysis produced %d findings gracefully", len(vulns))
}

func TestAnalyzeGraph_NilSafety(t *testing.T) {
	findings := AnalyzeGraph(nil)
	if len(findings) != 0 {
		t.Fatalf("expected nil graph to yield 0 findings, got %d", len(findings))
	}
}

func TestAnalyzeReport(t *testing.T) {
	source := []byte(`
const id = req.query.id;
db.query("SELECT * FROM users WHERE id = " + id);
`)
	report, err := AnalyzeReport("scan-123", "app.js", source)
	if err != nil {
		t.Fatalf("AnalyzeReport failed: %v", err)
	}
	if report.ScanID != "scan-123" {
		t.Errorf("expected ScanID 'scan-123', got %s", report.ScanID)
	}
	if report.FilePath != "app.js" {
		t.Errorf("expected FilePath 'app.js', got %s", report.FilePath)
	}
	if len(report.Vulnerabilities) == 0 {
		t.Fatalf("expected report to contain vulnerabilities")
	}
	if report.Timestamp.IsZero() {
		t.Errorf("expected report timestamp to be set")
	}
}
