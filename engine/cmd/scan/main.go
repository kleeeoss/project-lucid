package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lucid-ci/engine"
	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
)

type FindingResult struct {
	RuleID          string              `json:"rule_id"`
	RuleName        string              `json:"rule_name"`
	CWE             string              `json:"cwe"`
	Severity        string              `json:"severity"`
	ConfidenceScore float64             `json:"confidence_score"`
	FilePath        string              `json:"file_path"`
	LineStart       int                 `json:"line_start"`
	LineEnd         int                 `json:"line_end"`
	VulnerableCode  string              `json:"vulnerable_code"`
	ASTGraph        *models.GraphExport `json:"ast_graph,omitempty"`
}

func main() {
	jsonOutput := flag.Bool("json", false, "Output findings as JSON")
	flag.Parse()

	targetDir := "."
	if flag.NArg() > 0 {
		targetDir = flag.Arg(0)
	}

	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	var allFindings []FindingResult

	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		".venv":        true,
		"venv":         true,
		"vendor":       true,
		"__pycache__":  true,
		".next":        true,
		"dist":         true,
		"build":        true,
		".idea":        true,
	}

	err = filepath.Walk(absTarget, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".js" && ext != ".ts" && ext != ".py" && ext != ".jsx" {
			return nil
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil || len(source) == 0 {
			return nil
		}

		relPath, relErr := filepath.Rel(absTarget, path)
		if relErr != nil {
			relPath = path
		}

		g, normErr := graph.NormalizeSource(relPath, source)
		if normErr != nil {
			return nil
		}

		vulns := engine.AnalyzeGraph(g)
		for _, v := range vulns {
			rfExport := graph.ExportReactFlow(g, &v)
			allFindings = append(allFindings, FindingResult{
				RuleID:          v.RuleID,
				RuleName:        v.RuleName,
				CWE:             v.CWE,
				Severity:        string(v.Severity),
				ConfidenceScore: v.ConfidenceScore,
				FilePath:        relPath,
				LineStart:       v.LineStart,
				LineEnd:         v.LineEnd,
				VulnerableCode:  v.VulnerableCode,
				ASTGraph:        &rfExport,
			})
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Walk error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		if allFindings == nil {
			allFindings = []FindingResult{}
		}
		data, marshalErr := json.MarshalIndent(allFindings, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", marshalErr)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Human-readable output
	fmt.Printf("Lucid-CI Directory Scan: %s\n", absTarget)
	fmt.Printf("Total Findings Detected: %d\n\n", len(allFindings))
	for i, f := range allFindings {
		fmt.Printf("[%d] %s (%s) - %s\n", i+1, f.RuleID, f.CWE, f.Severity)
		fmt.Printf("    File: %s:%d-%d\n", f.FilePath, f.LineStart, f.LineEnd)
		fmt.Printf("    Code: %s\n", strings.TrimSpace(f.VulnerableCode))
		if f.ASTGraph != nil {
			fmt.Printf("    AST Flow: %d nodes, %d edges\n", len(f.ASTGraph.Nodes), len(f.ASTGraph.Edges))
		}
		fmt.Println()
	}
}
