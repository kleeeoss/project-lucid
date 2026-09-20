package rules

import "testing"

func TestDetectSQLInjectionJavaScriptVulnerableConcat(t *testing.T) {
	source := []byte(`function handler(req) {
  const id = req.query.id;
  const query = "SELECT * FROM users WHERE id = " + id;
  db.query(query);
}`)
	vulns, err := DetectSQLInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectSQLInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one SQLi finding, got %d: %+v", len(vulns), vulns)
	}
	v := vulns[0]
	if v.RuleID != SQLInjectionRuleID || v.CWE != SQLInjectionCWE || v.FilePath != "app.js" {
		t.Fatalf("unexpected vulnerability metadata: %+v", v)
	}
	if v.SourceNode.Name != "id" || v.SinkNode.Name != "db.query" || len(v.TaintPath) < 3 {
		t.Fatalf("unexpected taint path: %+v", v)
	}
	if v.ConfidenceScore <= 0 || v.ConfidenceScore > 1 {
		t.Fatalf("confidence out of range: %f", v.ConfidenceScore)
	}
}

func TestDetectSQLInjectionJavaScriptParameterizedQueryIsClean(t *testing.T) {
	source := []byte(`function handler(req) {
  const id = req.query.id;
  db.query("SELECT * FROM users WHERE id = ?", [id]);
}`)
	vulns, err := DetectSQLInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectSQLInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("parameterized query should not produce finding: %+v", vulns)
	}
}

func TestDetectSQLInjectionJavaScriptSanitizedIsClean(t *testing.T) {
	source := []byte(`function handler(req) {
  const id = req.query.id;
  const safe = sanitize(id);
  const query = "SELECT * FROM users WHERE id = " + safe;
  db.query(query);
}`)
	vulns, err := DetectSQLInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectSQLInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("sanitized query should not produce finding: %+v", vulns)
	}
}

func TestDetectSQLInjectionPythonVulnerableConcat(t *testing.T) {
	source := []byte("def handler(request):\n    name = request.args.get('name')\n    query = 'SELECT * FROM users WHERE name = ' + name\n    cursor.execute(query)\n")
	vulns, err := DetectSQLInjection("views.py", source)
	if err != nil {
		t.Fatalf("DetectSQLInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one Python SQLi finding, got %d: %+v", len(vulns), vulns)
	}
	if vulns[0].SinkNode.Name != "cursor.execute" || vulns[0].RuleID != SQLInjectionRuleID {
		t.Fatalf("unexpected finding: %+v", vulns[0])
	}
}

func TestDetectSQLInjectionCleanConstantSQL(t *testing.T) {
	vulns, err := DetectSQLInjection("app.js", []byte("db.query('SELECT * FROM users');"))
	if err != nil {
		t.Fatalf("DetectSQLInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("constant SQL should not produce finding: %+v", vulns)
	}
}
