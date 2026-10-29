package fixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lucid-ci/engine"
	"lucid-ci/engine/models"
	"lucid-ci/engine/rules"
)

// ----------------------------------------------------------------------------
// 1. Python Language Constructs
// ----------------------------------------------------------------------------

func TestCorpus_PythonConstructs(t *testing.T) {
	t.Run("fstrings_vulnerable_sqli", func(t *testing.T) {
		defer assertNoPanic(t, "python fstring vulnerable")
		src := []byte(`
name = request.args.get('name')
q = f"SELECT * FROM users WHERE name = '{name}'"
cursor.execute(q)
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 1)
	})

	t.Run("fstrings_constant_safe", func(t *testing.T) {
		defer assertNoPanic(t, "python fstring constant")
		src := []byte(`
limit = 50
q = f"SELECT * FROM users LIMIT {limit}"
cursor.execute(q)
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 0)
	})

	t.Run("fstrings_parameterized_safe", func(t *testing.T) {
		defer assertNoPanic(t, "python parameterized query")
		src := []byte(`
name = request.args.get('name')
cursor.execute("SELECT * FROM users WHERE name = %s", (name,))
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 0)
	})

	t.Run("nested_comprehensions", func(t *testing.T) {
		defer assertNoPanic(t, "python nested comprehensions")
		src := []byte(`
matrix = [[1, 2], [3, 4]]
flattened = [val for row in matrix for val in row if val > 1]
nested_dict = {k: {x: x * 2 for x in range(3)} for k in ['a', 'b']}
set_comp = {x * y for x in range(3) for y in range(3)}
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(vulns) != 0 {
			t.Fatalf("expected 0 vulns in clean comprehensions, got %d", len(vulns))
		}
	})

	t.Run("decorator_chains", func(t *testing.T) {
		defer assertNoPanic(t, "python decorator chains")
		src := []byte(`
@login_required
@limiter.limit("5/minute")
@cache.memoize(timeout=60)
def route_handler(request):
    return "ok"
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(vulns) != 0 {
			t.Fatalf("expected 0 vulns in decorated handler, got %d", len(vulns))
		}
	})

	t.Run("raw_sql_vs_orm", func(t *testing.T) {
		defer assertNoPanic(t, "python raw SQL vs ORM")
		// 1. Raw SQL: Vulnerable
		rawSrc := []byte(`
id_val = request.args.get('id')
cursor.execute("SELECT * FROM users WHERE id = " + id_val)
`)
		rawVulns, err := engine.Analyze("views.py", rawSrc)
		if err != nil {
			t.Fatalf("analyze raw: %v", err)
		}
		assertHasRule(t, rawVulns, rules.SQLInjectionRuleID, 1)

		// 2. ORM queries: Clean (not raw SQL sinks)
		ormSrc := []byte(`
id_val = request.args.get('id')
user = User.objects.filter(id=id_val).first()
records = session.query(Item).filter_by(user_id=id_val).all()
`)
		ormVulns, err := engine.Analyze("views.py", ormSrc)
		if err != nil {
			t.Fatalf("analyze ORM: %v", err)
		}
		assertHasRule(t, ormVulns, rules.SQLInjectionRuleID, 0)
	})

	t.Run("os_system_vs_shlex_quote", func(t *testing.T) {
		defer assertNoPanic(t, "python os.system vs shlex.quote")
		// Vulnerable without quote
		vulnSrc := []byte(`
host = request.args.get('host')
os.system("ping " + host)
`)
		vulnRes, err := engine.Analyze("views.py", vulnSrc)
		if err != nil {
			t.Fatalf("analyze vuln: %v", err)
		}
		assertHasRule(t, vulnRes, rules.CommandInjectionRuleID, 1)

		// Sanitized with shlex.quote
		safeSrc := []byte(`
import shlex
host = request.args.get('host')
safe_host = shlex.quote(host)
os.system("ping " + safe_host)
`)
		safeRes, err := engine.Analyze("views.py", safeSrc)
		if err != nil {
			t.Fatalf("analyze safe: %v", err)
		}
		assertHasRule(t, safeRes, rules.CommandInjectionRuleID, 0)
	})

	t.Run("match_case_and_async_generators", func(t *testing.T) {
		defer assertNoPanic(t, "python 3.10+ match case and async gen")
		src := []byte(`
def check(val):
    match val:
        case 1:
            return "one"
        case [x, y]:
            return x + y
        case _:
            return "other"

async def event_generator():
    for i in range(10):
        yield i
`)
		vulns, err := engine.Analyze("views.py", src)
		if err != nil {
			t.Fatalf("analyze match-case: %v", err)
		}
		if len(vulns) != 0 {
			t.Fatalf("expected 0 findings, got %d", len(vulns))
		}
	})
}

// ----------------------------------------------------------------------------
// 2. JavaScript & TypeScript Language Constructs
// ----------------------------------------------------------------------------

func TestCorpus_JavaScriptAndTypeScriptConstructs(t *testing.T) {
	t.Run("template_literals_vulnerable_sqli", func(t *testing.T) {
		defer assertNoPanic(t, "js template literal vulnerable")
		src := []byte("const id = req.query.id;\nconst q = `SELECT * FROM users WHERE id = ${id}`;\ndb.query(q);\n")
		vulns, err := engine.Analyze("app.js", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 1)
	})

	t.Run("template_literals_constant_safe", func(t *testing.T) {
		defer assertNoPanic(t, "js template literal constant safe")
		src := []byte("db.query(`SELECT * FROM users WHERE active = 1`);\n")
		vulns, err := engine.Analyze("app.js", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 0)
	})

	t.Run("tagged_templates_sanitized", func(t *testing.T) {
		defer assertNoPanic(t, "js tagged template sanitized")
		src := []byte("const id = req.query.id;\nconst q = sql`SELECT * FROM users WHERE id = ${id}`;\ndb.query(q);\n")
		vulns, err := engine.Analyze("app.js", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 0)
	})

	t.Run("dynamic_import", func(t *testing.T) {
		defer assertNoPanic(t, "js dynamic import")
		src := []byte(`
async function loadPlugin(req) {
  const mod = req.query.plugin;
  const loaded = await import(mod);
  return loaded;
}
`)
		vulns, err := engine.Analyze("app.js", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(vulns) != 0 {
			t.Fatalf("expected 0 vulns, got %d", len(vulns))
		}
	})

	t.Run("async_await_arrow_functions", func(t *testing.T) {
		defer assertNoPanic(t, "js async arrow functions")
		src := []byte(`
const handler = async (req, res) => {
  const host = req.query.host;
  const cmd = "ping " + host;
  child_process.exec(cmd);
};
`)
		vulns, err := engine.Analyze("app.js", src)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		assertHasRule(t, vulns, rules.CommandInjectionRuleID, 1)
	})

	t.Run("path_joins_vulnerable_vs_sanitized", func(t *testing.T) {
		defer assertNoPanic(t, "js path join vs sanitized")
		// 1. Vulnerable path.join without sanitization
		vulnSrc := []byte(`
const file = req.query.file;
const target = path.join("/var/uploads", file);
fs.readFile(target);
`)
		vulnRes, err := engine.Analyze("app.js", vulnSrc)
		if err != nil {
			t.Fatalf("analyze vuln: %v", err)
		}
		assertHasRule(t, vulnRes, rules.PathTraversalRuleID, 1)

		// 2. Sanitized with path.resolve
		resolveSrc := []byte(`
const file = req.query.file;
const target = path.resolve("/var/uploads", file);
fs.readFile(target);
`)
		resolveRes, err := engine.Analyze("app.js", resolveSrc)
		if err != nil {
			t.Fatalf("analyze resolve: %v", err)
		}
		assertHasRule(t, resolveRes, rules.PathTraversalRuleID, 0)

		// 3. Sanitized with path.normalize
		normSrc := []byte(`
const file = req.query.file;
const target = path.normalize(path.join("/var/uploads", file));
fs.readFile(target);
`)
		normRes, err := engine.Analyze("app.js", normSrc)
		if err != nil {
			t.Fatalf("analyze normalize: %v", err)
		}
		assertHasRule(t, normRes, rules.PathTraversalRuleID, 0)
	})

	t.Run("typescript_constructs", func(t *testing.T) {
		defer assertNoPanic(t, "typescript constructs")
		src := []byte(`
const findUser = async (req) => {
  const id = req.query.id;
  const sql = "SELECT * FROM users WHERE id = " + id;
  db.query(sql);
};
`)
		vulns, err := engine.Analyze("service.ts", src)
		if err != nil {
			t.Fatalf("analyze typescript: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 1)
	})
}

// ----------------------------------------------------------------------------
// 3. Edge Cases & Malformed Syntax
// ----------------------------------------------------------------------------

func TestCorpus_EdgeCasesAndMalformedSyntax(t *testing.T) {
	t.Run("incomplete_expressions_js", func(t *testing.T) {
		defer assertNoPanic(t, "incomplete js expressions")
		src := []byte(`
const x = ;
let y = 
function test( {
if (
`)
		vulns, err := engine.Analyze("broken.js", src)
		if err != nil {
			t.Fatalf("incomplete expression should not return hard error: %v", err)
		}
		t.Logf("incomplete expressions returned %d findings", len(vulns))
	})

	t.Run("incomplete_expressions_python", func(t *testing.T) {
		defer assertNoPanic(t, "incomplete python expressions")
		src := []byte(`
def broken(
    x = 
    while
`)
		vulns, err := engine.Analyze("broken.py", src)
		if err != nil {
			t.Fatalf("incomplete expression should not return hard error: %v", err)
		}
		t.Logf("incomplete python returned %d findings", len(vulns))
	})

	t.Run("syntax_errors_and_garbage_tokens", func(t *testing.T) {
		defer assertNoPanic(t, "garbage tokens")
		garbage := []byte(`
@@@###$$$%%%^^^&&&***(((
}}}}]]{{{{((((
function !!! ??? ~~~
`)
		vulns, err := engine.Analyze("garbage.js", garbage)
		if err != nil {
			t.Fatalf("garbage tokens should not crash parser: %v", err)
		}
		t.Logf("garbage source produced %d findings", len(vulns))
	})

	t.Run("truncated_file_javascript", func(t *testing.T) {
		defer assertNoPanic(t, "truncated js")
		full := `
const id = req.query.id;
const query = "SELECT * FROM users WHERE id = " + id;
db.query(query);
`
		for cut := 5; cut < len(full); cut += 10 {
			truncated := []byte(full[:cut])
			_, err := engine.Analyze("truncated.js", truncated)
			if err != nil {
				t.Fatalf("truncated js at byte %d threw error: %v", cut, err)
			}
		}
	})

	t.Run("truncated_file_python", func(t *testing.T) {
		defer assertNoPanic(t, "truncated python")
		full := `
name = request.args.get('name')
q = 'SELECT * FROM users WHERE name = ' + name
cursor.execute(q)
`
		for cut := 5; cut < len(full); cut += 10 {
			truncated := []byte(full[:cut])
			_, err := engine.Analyze("truncated.py", truncated)
			if err != nil {
				t.Fatalf("truncated python at byte %d threw error: %v", cut, err)
			}
		}
	})

	t.Run("large_block_2500_loc_javascript", func(t *testing.T) {
		defer assertNoPanic(t, "large js block")
		var b strings.Builder
		for i := 0; i < 2500; i++ {
			fmt.Fprintf(&b, "const blockVar%d = %d;\n", i, i)
		}
		b.WriteString("const id = req.query.id;\n")
		b.WriteString("const q = \"SELECT * FROM users WHERE id = \" + id;\n")
		b.WriteString("db.query(q);\n")

		vulns, err := engine.Analyze("large.js", []byte(b.String()))
		if err != nil {
			t.Fatalf("analyze 2500 LOC JS: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 1)
	})

	t.Run("large_block_2500_loc_python", func(t *testing.T) {
		defer assertNoPanic(t, "large python block")
		var b strings.Builder
		for i := 0; i < 2500; i++ {
			fmt.Fprintf(&b, "block_var_%d = %d\n", i, i)
		}
		b.WriteString("name = request.args.get('name')\n")
		b.WriteString("q = 'SELECT * FROM users WHERE name = ' + name\n")
		b.WriteString("cursor.execute(q)\n")

		vulns, err := engine.Analyze("large.py", []byte(b.String()))
		if err != nil {
			t.Fatalf("analyze 2500 LOC Python: %v", err)
		}
		assertHasRule(t, vulns, rules.SQLInjectionRuleID, 1)
	})
}

// ----------------------------------------------------------------------------
// 4. Corpus Micro-Repositories Directory Test
// ----------------------------------------------------------------------------

func TestCorpus_DirectoryMicroRepos(t *testing.T) {
	corpusDir := filepath.Join(".", "corpus")
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}
	if len(entries) < 10 {
		t.Fatalf("expected at least 10 sample files in corpus dir, found %d", len(entries))
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		filePath := filepath.Join(corpusDir, fileName)

		t.Run(fileName, func(t *testing.T) {
			defer assertNoPanic(t, "corpus file: "+fileName)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("read file %q: %v", fileName, err)
			}

			vulns, err := engine.Analyze(fileName, content)
			if err != nil {
				t.Fatalf("analyze corpus file %q: %v", fileName, err)
			}

			t.Logf("file %s -> %d findings", fileName, len(vulns))
			if strings.Contains(fileName, "vuln") {
				if len(vulns) == 0 {
					t.Errorf("vulnerable sample %s expected findings, got 0", fileName)
				}
			}
			if strings.Contains(fileName, "safe") {
				if len(vulns) != 0 {
					t.Errorf("safe sample %s expected 0 findings, got %d: %+v", fileName, len(vulns), vulns)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Test Helpers
// ----------------------------------------------------------------------------

func assertNoPanic(t *testing.T, context string) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("PANIC in %s: %v", context, r)
	}
}

func assertHasRule(t *testing.T, vulns []models.Vulnerability, expectedRuleID string, expectedCount int) {
	t.Helper()
	actual := 0
	for _, v := range vulns {
		if v.RuleID == expectedRuleID {
			actual++
		}
	}
	if actual != expectedCount {
		t.Fatalf("expected %d findings for %s, got %d (total: %d, vulns: %+v)", expectedCount, expectedRuleID, actual, len(vulns), vulns)
	}
}
