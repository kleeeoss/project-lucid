package fixtures

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/rules"
)

type fixtureCase struct {
	name        string
	filePath    string
	source      string
	wantSQLi    int
	wantCommand int
}

func TestSyntheticSecuritySuite(t *testing.T) {
	cases := []fixtureCase{
		{"js_sqli_concat", "app.js", `const id = req.query.id; const q = "SELECT * FROM users WHERE id = " + id; db.query(q);`, 1, 0},
		{"js_sqli_transitive", "app.js", `const a = req.body.id; const b = a; const q = "SELECT * FROM users WHERE id = " + b; db.query(q);`, 1, 0},
		{"js_sqli_parameterized", "app.js", `const id = req.query.id; db.query("SELECT * FROM users WHERE id = ?", [id]);`, 0, 0},
		{"js_sqli_sanitized", "app.js", `const id = req.query.id; const safe = sanitize(id); const q = "SELECT * FROM users WHERE id = " + safe; db.query(q);`, 0, 0},
		{"js_sqli_constant", "app.js", `db.query("SELECT * FROM users");`, 0, 0},
		{"py_sqli_concat", "views.py", "name = request.args.get('name')\nq = 'SELECT * FROM users WHERE name = ' + name\ncursor.execute(q)\n", 1, 0},
		{"py_sqli_transitive", "views.py", "a = request.form.get('id')\nb = a\nq = 'SELECT * FROM users WHERE id = ' + b\ncursor.execute(q)\n", 1, 0},
		{"py_sqli_parameterized", "views.py", "name = request.args.get('name')\ncursor.execute('SELECT * FROM users WHERE name = %s', (name,))\n", 0, 0},
		{"py_sqli_sanitized", "views.py", "name = request.args.get('name')\nsafe = sanitize(name)\nq = 'SELECT * FROM users WHERE name = ' + safe\ncursor.execute(q)\n", 0, 0},
		{"py_sqli_constant", "views.py", "cursor.execute('SELECT * FROM users')\n", 0, 0},
		{"js_cmd_exec", "app.js", `const host = req.query.host; const cmd = "ping " + host; child_process.exec(cmd);`, 0, 1},
		{"js_cmd_execsync", "app.js", `const arg = req.body.arg; execSync("ls " + arg);`, 0, 1},
		{"js_cmd_transitive", "app.js", `const a = req.params.name; const b = a; const cmd = "cat " + b; child_process.exec(cmd);`, 0, 1},
		{"js_cmd_sanitized", "app.js", `const host = req.query.host; const safe = escape(host); child_process.exec("ping " + safe);`, 0, 0},
		{"js_cmd_constant", "app.js", `child_process.exec("uptime");`, 0, 0},
		{"py_cmd_os_system", "views.py", "target = request.args.get('target')\ncmd = 'ping ' + target\nos.system(cmd)\n", 0, 1},
		{"py_cmd_popen_shell_true", "views.py", "arg = request.args.get('arg')\nsubprocess.Popen('grep ' + arg, shell=True)\n", 0, 1},
		{"py_cmd_popen_shell_false", "views.py", "arg = request.args.get('arg')\nsubprocess.Popen(['grep', arg], shell=False)\n", 0, 0},
		{"py_cmd_sanitized", "views.py", "arg = request.args.get('arg')\nsafe = escape(arg)\nos.system('grep ' + safe)\n", 0, 0},
		{"py_cmd_constant", "views.py", "os.system('uptime')\n", 0, 0},
		{"js_clean_user_value", "app.js", `const id = user.id; db.query(id);`, 0, 0},
		{"py_clean_local_value", "views.py", "name = 'alice'\ncursor.execute(name)\n", 0, 0},
		{"js_malformed_no_panic", "broken.js", `function broken( { const id = req.query.id; db.query(id);`, 0, 0},
		{"py_malformed_no_panic", "broken.py", "def broken(:\n    x = request.args.get('x')\n", 0, 0},
		{"js_dual_sqli_cmd", "app.js", `const id = req.query.id; const q = "SELECT " + id; db.query(q); const host = req.query.host; child_process.exec("ping " + host);`, 1, 1},
	}
	if len(cases) != 25 {
		t.Fatalf("fixture suite must contain exactly 25 cases, got %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqli, err := rules.DetectSQLInjection(tc.filePath, []byte(tc.source))
			if err != nil {
				t.Fatalf("SQLi detector error: %v", err)
			}
			cmd, err := rules.DetectCommandInjection(tc.filePath, []byte(tc.source))
			if err != nil {
				t.Fatalf("command detector error: %v", err)
			}
			if len(sqli) != tc.wantSQLi || len(cmd) != tc.wantCommand {
				t.Fatalf("got SQLi=%d Command=%d, want SQLi=%d Command=%d", len(sqli), len(cmd), tc.wantSQLi, tc.wantCommand)
			}
		})
	}
}

func TestNormalizeAndTaint500LinesUnderBudget(t *testing.T) {
	source := syntheticJavaScript500LOC()
	warmGraph, err := graph.NormalizeSource("large.js", []byte(source))
	if err != nil {
		t.Fatalf("warm normalize 500 LOC: %v", err)
	}
	_ = graph.PropagateTaint(warmGraph, rules.NewCatalog())

	start := time.Now()
	g, err := graph.NormalizeSource("large.js", []byte(source))
	if err != nil {
		t.Fatalf("normalize 500 LOC: %v", err)
	}
	_ = graph.PropagateTaint(g, rules.NewCatalog())
	elapsed := time.Since(start)
	budget := 50 * time.Millisecond
	if raceEnabled || testing.CoverMode() != "" {
		budget = 500 * time.Millisecond
	}
	if elapsed > budget {
		t.Fatalf("500 LOC traversal exceeded %s budget: %s", budget, elapsed)
	}
}

func syntheticJavaScript500LOC() string {
	var b strings.Builder
	for i := 0; i < 497; i++ {
		fmt.Fprintf(&b, "// filler line %d\n", i)
	}
	b.WriteString("const id = req.query.id;\n")
	b.WriteString("const q = \"SELECT * FROM users WHERE id = \" + id;\n")
	b.WriteString("db.query(q);\n")
	return b.String()
}
