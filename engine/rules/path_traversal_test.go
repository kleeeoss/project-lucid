package rules

import "testing"

func TestDetectPathTraversalJavaScriptReadFile(t *testing.T) {
	source := []byte(`function handler(req) {
  const name = req.query.file;
  const target = "uploads/" + name;
  fs.readFile(target);
}`)
	vulns, err := DetectPathTraversal("app.js", source)
	if err != nil {
		t.Fatalf("DetectPathTraversal: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected path traversal finding, got %+v", vulns)
	}
	if vulns[0].RuleID != PathTraversalRuleID || vulns[0].CWE != PathTraversalCWE || vulns[0].SinkNode.Name != "fs.readFile" {
		t.Fatalf("unexpected finding: %+v", vulns[0])
	}
}

func TestDetectPathTraversalPythonOpen(t *testing.T) {
	source := []byte("def handler(request):\n    name = request.args.get('file')\n    target = 'uploads/' + name\n    open(target)\n")
	vulns, err := DetectPathTraversal("views.py", source)
	if err != nil {
		t.Fatalf("DetectPathTraversal: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected Python path traversal finding, got %+v", vulns)
	}
	if vulns[0].SinkNode.Name != "open" {
		t.Fatalf("unexpected sink: %+v", vulns[0])
	}
}

func TestDetectPathTraversalBoundaryCheckIsClean(t *testing.T) {
	source := []byte(`function handler(req) {
  const name = req.query.file;
  const target = path.resolve("uploads", name);
  fs.readFile(target);
}`)
	vulns, err := DetectPathTraversal("app.js", source)
	if err != nil {
		t.Fatalf("DetectPathTraversal: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("boundary-checked path should not produce finding: %+v", vulns)
	}
}

func TestDetectPathTraversalSanitizedIsClean(t *testing.T) {
	source := []byte("def handler(request):\n    name = request.args.get('file')\n    safe = os.path.abspath(name)\n    open(safe)\n")
	vulns, err := DetectPathTraversal("views.py", source)
	if err != nil {
		t.Fatalf("DetectPathTraversal: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("sanitized path should not produce finding: %+v", vulns)
	}
}

func TestDetectPathTraversalCleanConstantPath(t *testing.T) {
	vulns, err := DetectPathTraversal("app.js", []byte(`fs.readFile("uploads/readme.txt");`))
	if err != nil {
		t.Fatalf("DetectPathTraversal: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("constant path should not produce finding: %+v", vulns)
	}
}

func TestDetectPathTraversalJavaScriptReadFileSync(t *testing.T) {
	source := []byte(`const express = require("express");
const app = express();
app.get("/download", (req, res) => {
	const file = req.query.file;
	fs.readFileSync("/var/www/" + file);
});`)

	vulns, err := DetectPathTraversal("app.js", source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vulns) != 1 {
		t.Fatalf("expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].RuleID != PathTraversalRuleID {
		t.Fatalf("expected rule %s, got %s", PathTraversalRuleID, vulns[0].RuleID)
	}

	if vulns[0].CWE != PathTraversalCWE {
		t.Fatalf("expected CWE %s, got %s", PathTraversalCWE, vulns[0].CWE)
	}

	if vulns[0].SinkNode.Name != "fs.readFileSync" {
		t.Fatalf("expected sink fs.readFileSync, got %s", vulns[0].SinkNode.Name)
	}
}
