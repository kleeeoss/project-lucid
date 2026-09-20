package rules

import "testing"

func TestDetectCommandInjectionJavaScriptExec(t *testing.T) {
	source := []byte(`function handler(req) {
  const host = req.query.host;
  const cmd = "ping " + host;
  child_process.exec(cmd);
}`)
	vulns, err := DetectCommandInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one command injection finding, got %d: %+v", len(vulns), vulns)
	}
	v := vulns[0]
	if v.RuleID != CommandInjectionRuleID || v.CWE != CommandInjectionCWE || v.Severity != "CRITICAL" {
		t.Fatalf("unexpected vulnerability metadata: %+v", v)
	}
	if v.SourceNode.Name != "host" || v.SinkNode.Name != "child_process.exec" || len(v.TaintPath) < 3 {
		t.Fatalf("unexpected taint path: %+v", v)
	}
}

func TestDetectCommandInjectionJavaScriptExecSync(t *testing.T) {
	source := []byte(`function handler(req) {
  const arg = req.body.arg;
  execSync("ls " + arg);
}`)
	vulns, err := DetectCommandInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected execSync finding, got %+v", vulns)
	}
}

func TestDetectCommandInjectionPythonOSSystem(t *testing.T) {
	source := []byte("def handler(request):\n    target = request.args.get('target')\n    cmd = 'ping ' + target\n    os.system(cmd)\n")
	vulns, err := DetectCommandInjection("views.py", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected os.system finding, got %+v", vulns)
	}
	if vulns[0].SinkNode.Name != "os.system" {
		t.Fatalf("unexpected sink: %+v", vulns[0])
	}
}

func TestDetectCommandInjectionPythonPopenShellTrue(t *testing.T) {
	source := []byte("def handler(request):\n    arg = request.args.get('arg')\n    subprocess.Popen('grep ' + arg, shell=True)\n")
	vulns, err := DetectCommandInjection("views.py", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected subprocess.Popen shell=True finding, got %+v", vulns)
	}
}

func TestDetectCommandInjectionPythonPopenShellFalseIsClean(t *testing.T) {
	source := []byte("def handler(request):\n    arg = request.args.get('arg')\n    subprocess.Popen(['grep', arg], shell=False)\n")
	vulns, err := DetectCommandInjection("views.py", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("shell=False call should not produce finding: %+v", vulns)
	}
}

func TestDetectCommandInjectionSanitizedIsClean(t *testing.T) {
	source := []byte(`function handler(req) {
  const host = req.query.host;
  const safe = escape(host);
  child_process.exec("ping " + safe);
}`)
	vulns, err := DetectCommandInjection("app.js", source)
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("sanitized command should not produce finding: %+v", vulns)
	}
}

func TestDetectCommandInjectionCleanConstantCommand(t *testing.T) {
	vulns, err := DetectCommandInjection("app.js", []byte("child_process.exec('uptime');"))
	if err != nil {
		t.Fatalf("DetectCommandInjection: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("constant command should not produce finding: %+v", vulns)
	}
}
