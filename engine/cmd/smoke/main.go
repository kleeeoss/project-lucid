package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"lucid-ci/engine/parser"
)

func main() {
	source := []byte(`
const express = require("express");
app.get("/user", (req, res) => {
  const id = req.query.id;
  const query = "SELECT * FROM users WHERE id = " + id;
  db.query(query);
});
`)

	p, err := parser.NewJavaScriptParser()
	if err != nil {
		log.Fatalf("create parser: %v", err)
	}
	defer p.Close()

	start := time.Now()
	tree, err := p.Parse(source)
	if err != nil {
		log.Fatalf("parse smoke source: %v", err)
	}
	defer tree.Close()
	duration := time.Since(start)

	root, err := tree.Root()
	if err != nil {
		log.Fatalf("read root: %v", err)
	}
	fmt.Printf("language=javascript root=%s has_error=%t parse_ms=%.3f\n", root.Kind, root.HasError, float64(duration.Microseconds())/1000.0)

	nodes, err := tree.WalkNamed(4)
	if err != nil {
		log.Fatalf("walk tree: %v", err)
	}
	for _, node := range nodes {
		indent := strings.Repeat("  ", node.Depth)
		fmt.Printf("%s%s [%d:%d-%d:%d] %q\n", indent, node.Kind, node.StartLine, node.StartColumn, node.EndLine, node.EndColumn, node.Snippet)
	}

	if duration > 50*time.Millisecond {
		log.Fatalf("parse exceeded 50ms budget: %s", duration)
	}
}
