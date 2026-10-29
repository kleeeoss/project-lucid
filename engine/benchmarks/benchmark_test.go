package benchmarks

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"lucid-ci/engine"
	"lucid-ci/engine/graph"
	"lucid-ci/engine/parser"
	"lucid-ci/engine/rules"
)

// ----------------------------------------------------------------------------
// Synthetic Code Samples
// ----------------------------------------------------------------------------

const jsSyntheticSQLi = `
const id = req.query.id;
const q = "SELECT * FROM users WHERE id = " + id;
db.query(q);
`

const jsSyntheticCmd = `
const host = req.query.host;
const cmd = "ping " + host;
child_process.exec(cmd);
`

const jsSyntheticPath = `
const file = req.query.file;
const target = path.join("/uploads", file);
fs.readFile(target);
`

const jsSyntheticSecrets = `
const apiKey = "AKIA1234567890ABCDEF";
const githubToken = "ghp_123456789012345678901234567890123456";
`

const pySyntheticSQLi = `
name = request.args.get('name')
q = 'SELECT * FROM users WHERE name = ' + name
cursor.execute(q)
`

const pySyntheticCmd = `
target = request.args.get('target')
cmd = 'ping ' + target
os.system(cmd)
`

const pySyntheticPath = `
filename = request.args.get('path')
p = "/data/" + filename
open(p)
`

const pySyntheticSecrets = `
aws_key = "AKIA9876543210ZYXWVU"
secret_token = "z9Aq7LmN4pR8xT2vB6yC0dE3fG5hJ7kL"
`

// ----------------------------------------------------------------------------
// Realistic Code Samples
// ----------------------------------------------------------------------------

const jsRealisticExpress = `
const express = require("express");
const path = require("path");
const fs = require("fs");
const { exec } = require("child_process");
const db = require("./db");

const app = express();
app.use(express.json());

app.get("/api/users", (req, res) => {
  const userId = req.query.id;
  const query = "SELECT id, username, email FROM users WHERE id = " + userId;
  db.query(query, (err, rows) => {
    if (err) return res.status(500).json({ error: "db error" });
    res.json(rows);
  });
});

app.post("/api/backup", (req, res) => {
  const table = req.body.table;
  exec("mysqldump -u root db " + table, (err, stdout) => {
    res.send(stdout);
  });
});

app.get("/api/docs", (req, res) => {
  const doc = req.query.doc;
  fs.readFile(path.join("/var/docs", doc), (err, data) => {
    res.send(data);
  });
});
`

const pyRealisticFlask = `
from flask import Flask, request, jsonify
import os
import subprocess
import sqlite3

app = Flask(__name__)

@app.route("/search")
def search():
    term = request.args.get("q", "")
    conn = sqlite3.connect("app.db")
    cursor = conn.cursor()
    query = "SELECT * FROM items WHERE name = '" + term + "'"
    cursor.execute(query)
    return jsonify(cursor.fetchall())

@app.route("/ping")
def ping():
    host = request.args.get("host", "127.0.0.1")
    cmd = "ping -c 1 " + host
    os.system(cmd)
    return jsonify({"status": "pinged"})

@app.route("/view")
def view_file():
    filename = request.args.get("file", "default.txt")
    with open("/app/data/" + filename) as f:
        return f.read()
`

// ----------------------------------------------------------------------------
// 1. Complete Analysis Flow Benchmarks (engine.Analyze)
// ----------------------------------------------------------------------------

func BenchmarkAnalyze_Synthetic_JS_SQLi(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsSyntheticSQLi)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("app.js", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected SQLi finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_JS_CommandInjection(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsSyntheticCmd)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("app.js", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected command injection finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_JS_PathTraversal(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsSyntheticPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("app.js", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected path traversal finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_JS_InsecureSecrets(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsSyntheticSecrets)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("app.js", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected secrets finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_PY_SQLi(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pySyntheticSQLi)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("views.py", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected SQLi finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_PY_CommandInjection(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pySyntheticCmd)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("views.py", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected command injection finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_PY_PathTraversal(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pySyntheticPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("views.py", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected path traversal finding")
		}
	}
}

func BenchmarkAnalyze_Synthetic_PY_InsecureSecrets(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pySyntheticSecrets)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("views.py", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected secrets finding")
		}
	}
}

func BenchmarkAnalyze_Realistic_Express_SQLi(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsRealisticExpress)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("server.js", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected findings in realistic express")
		}
	}
}

func BenchmarkAnalyze_Realistic_Flask_CommandInjection(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pyRealisticFlask)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("views.py", src)
		if err != nil {
			b.Fatalf("analyze: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected findings in realistic flask")
		}
	}
}

// ----------------------------------------------------------------------------
// 2. Individual Pipeline Stages Benchmarks
// ----------------------------------------------------------------------------

func BenchmarkStage_Parsing_JavaScript(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsRealisticExpress)
	p, err := parser.NewJavaScriptParser()
	if err != nil {
		b.Fatalf("parser: %v", err)
	}
	defer p.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree, err := p.Parse(src)
		if err != nil {
			b.Fatalf("parse: %v", err)
		}
		tree.Close()
	}
}

func BenchmarkStage_Parsing_Python(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pyRealisticFlask)
	p, err := parser.NewPythonParser()
	if err != nil {
		b.Fatalf("parser: %v", err)
	}
	defer p.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree, err := p.Parse(src)
		if err != nil {
			b.Fatalf("parse: %v", err)
		}
		tree.Close()
	}
}

func BenchmarkStage_Normalization_JavaScript(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsRealisticExpress)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err := graph.NormalizeSource("server.js", src)
		if err != nil {
			b.Fatalf("normalize: %v", err)
		}
		if len(g.Nodes) == 0 {
			b.Fatalf("expected nodes")
		}
	}
}

func BenchmarkStage_Normalization_Python(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pyRealisticFlask)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err := graph.NormalizeSource("views.py", src)
		if err != nil {
			b.Fatalf("normalize: %v", err)
		}
		if len(g.Nodes) == 0 {
			b.Fatalf("expected nodes")
		}
	}
}

func BenchmarkStage_TaintPropagation_JavaScript(b *testing.B) {
	b.ReportAllocs()
	src := []byte(jsRealisticExpress)
	g, err := graph.NormalizeSource("server.js", src)
	if err != nil {
		b.Fatalf("normalize: %v", err)
	}
	catalog := rules.NewCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = graph.PropagateTaint(g, catalog)
	}
}

func BenchmarkStage_TaintPropagation_Python(b *testing.B) {
	b.ReportAllocs()
	src := []byte(pyRealisticFlask)
	g, err := graph.NormalizeSource("views.py", src)
	if err != nil {
		b.Fatalf("normalize: %v", err)
	}
	catalog := rules.NewCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = graph.PropagateTaint(g, catalog)
	}
}

// ----------------------------------------------------------------------------
// 3. Latency Invariant Verification (< 50ms per 500 LOC)
// ----------------------------------------------------------------------------

func TestLatencyInvariant500LOC_JavaScript(t *testing.T) {
	source := []byte(generate500LOCJavaScript())

	// Warmup
	_, _ = engine.Analyze("bench500.js", source)

	start := time.Now()
	findings, err := engine.Analyze("bench500.js", source)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("analyze 500 LOC JS: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected vulnerability findings in 500 LOC JS sample")
	}

	budget := 50 * time.Millisecond
	if testing.CoverMode() != "" {
		budget = 500 * time.Millisecond
	}
	if elapsed > budget {
		t.Fatalf("500 LOC JS analysis exceeded %s budget: %s", budget, elapsed)
	}
	t.Logf("500 LOC JS analysis took %s (budget: %s, findings: %d)", elapsed, budget, len(findings))
}

func TestLatencyInvariant500LOC_Python(t *testing.T) {
	source := []byte(generate500LOCPython())

	// Warmup
	_, _ = engine.Analyze("bench500.py", source)

	start := time.Now()
	findings, err := engine.Analyze("bench500.py", source)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("analyze 500 LOC Python: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected vulnerability findings in 500 LOC Python sample")
	}

	budget := 50 * time.Millisecond
	if testing.CoverMode() != "" {
		budget = 500 * time.Millisecond
	}
	if elapsed > budget {
		t.Fatalf("500 LOC Python analysis exceeded %s budget: %s", budget, elapsed)
	}
	t.Logf("500 LOC Python analysis took %s (budget: %s, findings: %d)", elapsed, budget, len(findings))
}

func BenchmarkLatency500LOC_JavaScript(b *testing.B) {
	b.ReportAllocs()
	src := []byte(generate500LOCJavaScript())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("bench500.js", src)
		if err != nil {
			b.Fatalf("analyze 500 LOC JS: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected findings")
		}
	}
}

func BenchmarkLatency500LOC_Python(b *testing.B) {
	b.ReportAllocs()
	src := []byte(generate500LOCPython())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, err := engine.Analyze("bench500.py", src)
		if err != nil {
			b.Fatalf("analyze 500 LOC Python: %v", err)
		}
		if len(findings) == 0 {
			b.Fatalf("expected findings")
		}
	}
}

// ----------------------------------------------------------------------------
// Helpers: 500 LOC Synthetic Generators
// ----------------------------------------------------------------------------

func generate500LOCJavaScript() string {
	var b strings.Builder
	for i := 0; i < 490; i++ {
		fmt.Fprintf(&b, "const dummyVar%d = %d;\n", i, i)
	}
	b.WriteString("const id = req.query.id;\n")
	b.WriteString("const q = \"SELECT * FROM users WHERE id = \" + id;\n")
	b.WriteString("db.query(q);\n")
	b.WriteString("const target = req.query.host;\n")
	b.WriteString("child_process.exec(\"ping \" + target);\n")
	b.WriteString("const doc = req.query.file;\n")
	b.WriteString("fs.readFile(\"/data/\" + doc);\n")
	b.WriteString("const secret = \"AKIA1234567890ABCDEF\";\n")
	return b.String()
}

func generate500LOCPython() string {
	var b strings.Builder
	for i := 0; i < 490; i++ {
		fmt.Fprintf(&b, "dummy_var_%d = %d\n", i, i)
	}
	b.WriteString("name = request.args.get('name')\n")
	b.WriteString("q = 'SELECT * FROM users WHERE name = ' + name\n")
	b.WriteString("cursor.execute(q)\n")
	b.WriteString("host = request.args.get('host')\n")
	b.WriteString("os.system('ping ' + host)\n")
	b.WriteString("doc = request.args.get('file')\n")
	b.WriteString("open('/data/' + doc)\n")
	b.WriteString("key = 'ghp_123456789012345678901234567890123456'\n")
	return b.String()
}
