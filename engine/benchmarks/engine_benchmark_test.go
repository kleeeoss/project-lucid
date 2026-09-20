package benchmarks

import (
	"fmt"
	"strings"
	"testing"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/rules"
)

func BenchmarkNormalize500LOC(b *testing.B) {
	source := []byte(benchmarkJavaScript500LOC())
	for i := 0; i < b.N; i++ {
		g, err := graph.NormalizeSource("bench.js", source)
		if err != nil {
			b.Fatalf("normalize: %v", err)
		}
		if len(g.Nodes) == 0 {
			b.Fatalf("expected nodes")
		}
	}
}

func BenchmarkTaint500LOC(b *testing.B) {
	source := []byte(benchmarkJavaScript500LOC())
	g, err := graph.NormalizeSource("bench.js", source)
	if err != nil {
		b.Fatalf("normalize: %v", err)
	}
	catalog := rules.NewCatalog()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = graph.PropagateTaint(g, catalog)
	}
}

func benchmarkJavaScript500LOC() string {
	var b strings.Builder
	for i := 0; i < 497; i++ {
		fmt.Fprintf(&b, "const clean%d = %d;\n", i, i)
	}
	b.WriteString("const id = req.query.id;\n")
	b.WriteString("const q = \"SELECT * FROM users WHERE id = \" + id;\n")
	b.WriteString("db.query(q);\n")
	return b.String()
}
