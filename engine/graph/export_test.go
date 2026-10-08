package graph_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
	"lucid-ci/engine/rules"
)

const (
	wantActiveStyle     = "stroke: #ef4444; stroke-width: 2.5px;"
	wantStructuralStyle = "stroke: #94a3b8; stroke-width: 1px;"
)

func TestExportReactFlowNodeClassification(t *testing.T) {
	g, vuln := syntheticTaintGraph()
	export := graph.ExportReactFlow(g, &vuln)

	assertNodeType(t, export, "n_source", "sourceNode")
	assertNodeType(t, export, "n_taint", "taintNode")
	assertNodeType(t, export, "n_sink", "sinkNode")
	assertNodeType(t, export, "n_neutral", "astNode")
}

func TestExportReactFlowActiveTaintEdges(t *testing.T) {
	g, vuln := syntheticTaintGraph()
	export := graph.ExportReactFlow(g, &vuln)

	active := activeEdges(export)
	if len(active) != 2 {
		t.Fatalf("expected two active taint edges, got %d: %+v", len(active), active)
	}
	for _, edge := range active {
		if !edge.Animated || edge.Style != wantActiveStyle {
			t.Fatalf("active edge has wrong style: %+v", edge)
		}
	}
}

func TestExportReactFlowStructuralEdges(t *testing.T) {
	g, vuln := syntheticTaintGraph()
	export := graph.ExportReactFlow(g, &vuln)

	foundStructural := false
	for _, edge := range export.Edges {
		if edge.Source == "n_source" && edge.Target == "n_neutral" {
			foundStructural = true
			if edge.Animated || edge.Style != wantStructuralStyle {
				t.Fatalf("structural edge should be static slate: %+v", edge)
			}
		}
	}
	if !foundStructural {
		t.Fatalf("expected source -> neutral structural edge in export: %+v", export.Edges)
	}
}

func TestExportReactFlowDeterministicIDs(t *testing.T) {
	g, vuln := syntheticTaintGraph()
	first := graph.ExportReactFlow(g, &vuln)
	second := graph.ExportReactFlow(g, &vuln)

	if !reflect.DeepEqual(nodeIDs(first), nodeIDs(second)) {
		t.Fatalf("node IDs not deterministic: %v vs %v", nodeIDs(first), nodeIDs(second))
	}
	if !reflect.DeepEqual(edgeIDs(first), edgeIDs(second)) {
		t.Fatalf("edge IDs not deterministic: %v vs %v", edgeIDs(first), edgeIDs(second))
	}
}

func TestExportReactFlowDeterministicLayout(t *testing.T) {
	g, vuln := syntheticWideGraph(12)
	first := graph.ExportReactFlow(g, &vuln)
	second := graph.ExportReactFlow(g, &vuln)

	if !reflect.DeepEqual(nodePositions(first), nodePositions(second)) {
		t.Fatalf("node positions not deterministic: %+v vs %+v", nodePositions(first), nodePositions(second))
	}
}

func TestExportReactFlowCoordinateSafety(t *testing.T) {
	g, vuln := syntheticWideGraph(16)
	export := graph.ExportReactFlow(g, &vuln)

	seen := map[string]string{}
	for _, node := range export.Nodes {
		if node.Position.X < 0 || node.Position.Y < 0 {
			t.Fatalf("node has negative coordinate: %+v", node)
		}
		key := positionKey(node.Position)
		if prior, ok := seen[key]; ok {
			t.Fatalf("nodes %s and %s share coordinate %s", prior, node.ID, key)
		}
		seen[key] = node.ID
	}
}

func TestExportReactFlowNodeLimit(t *testing.T) {
	g, vuln := syntheticWideGraph(80)
	export := graph.ExportReactFlow(g, &vuln)
	if len(export.Nodes) > 50 {
		t.Fatalf("exported node count exceeded limit: %d", len(export.Nodes))
	}
	assertHasNode(t, export, "n_source")
	assertHasNode(t, export, "n_sink")
}

func TestExportReactFlowJSONRoundTrip(t *testing.T) {
	g, vuln := syntheticTaintGraph()
	export := graph.ExportReactFlow(g, &vuln)

	payload, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}
	var roundTrip models.GraphExport
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if len(roundTrip.Nodes) != len(export.Nodes) || len(roundTrip.Edges) != len(export.Edges) {
		t.Fatalf("round trip changed graph sizes: %+v vs %+v", roundTrip, export)
	}
}

func TestExportReactFlowSQLInjection(t *testing.T) {
	source := []byte(`const id = req.query.id;
const query = "SELECT * FROM users WHERE id = " + id;
db.query(query);`)
	g, err := graph.NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	vulns, err := rules.DetectSQLInjection("app.js", source)
	if err != nil {
		t.Fatalf("detect SQLi: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one SQLi vulnerability, got %+v", vulns)
	}
	export := graph.ExportReactFlow(g, &vulns[0])
	assertNodeType(t, export, nodeIDForRef(t, g, vulns[0].SourceNode), "sourceNode")
	assertNodeType(t, export, nodeIDForRef(t, g, vulns[0].SinkNode), "sinkNode")
	if len(activeEdges(export)) == 0 {
		t.Fatalf("expected active SQLi taint edges")
	}
}

func TestExportReactFlowCommandInjection(t *testing.T) {
	source := []byte(`const host = req.query.host;
const cmd = "ping " + host;
child_process.exec(cmd);`)
	g, err := graph.NormalizeSource("app.js", source)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	vulns, err := rules.DetectCommandInjection("app.js", source)
	if err != nil {
		t.Fatalf("detect command injection: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one command injection vulnerability, got %+v", vulns)
	}
	export := graph.ExportReactFlow(g, &vulns[0])
	assertNodeType(t, export, nodeIDForRef(t, g, vulns[0].SourceNode), "sourceNode")
	assertNodeType(t, export, nodeIDForRef(t, g, vulns[0].SinkNode), "sinkNode")
	if len(activeEdges(export)) == 0 {
		t.Fatalf("expected active command injection taint edges")
	}
}

func TestExportReactFlowComplexControlFlowDoesNotPanic(t *testing.T) {
	g, vuln := syntheticWideGraph(20)
	g.Edges = append(g.Edges,
		graph.Edge{ID: "branch:a", Source: "branch_parent", Target: "branch_left", Kind: "ast_child"},
		graph.Edge{ID: "branch:b", Source: "branch_parent", Target: "branch_right", Kind: "ast_child"},
	)
	g.Nodes = append(g.Nodes,
		node("branch_parent", graph.SemanticFunctionDeclaration, "handler", 200, 250, 10, 1, 1),
		node("branch_left", graph.SemanticVariableDeclaration, "left", 210, 220, 11, 3, 2),
		node("branch_right", graph.SemanticVariableDeclaration, "right", 221, 230, 12, 3, 2),
	)
	_ = graph.ExportReactFlow(g, &vuln)
}

func TestExportReactFlowNilInputs(t *testing.T) {
	if export := graph.ExportReactFlow(nil, nil); len(export.Nodes) != 0 || len(export.Edges) != 0 {
		t.Fatalf("nil graph should export empty graph: %+v", export)
	}
	g, _ := syntheticTaintGraph()
	if export := graph.ExportReactFlow(g, nil); len(export.Nodes) == 0 {
		t.Fatalf("nil vulnerability should still export graph context")
	}
}

func TestExportReactFlowLivePRFixtures(t *testing.T) {
	cases := []struct {
		name         string
		filePath     string
		detector     func(string, []byte) ([]models.Vulnerability, error)
		wantSinkLine int
	}{
		{name: "express_sql", filePath: filepath.Join("..", "fixtures", "live_pr_express.js"), detector: rules.DetectSQLInjection, wantSinkLine: 13},
		{name: "flask_command", filePath: filepath.Join("..", "fixtures", "live_pr_flask.py"), detector: rules.DetectCommandInjection, wantSinkLine: 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(tc.filePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			analysisPath := filepath.Base(tc.filePath)
			g, err := graph.NormalizeSource(analysisPath, source)
			if err != nil {
				t.Fatalf("normalize fixture: %v", err)
			}
			vulns, err := tc.detector(analysisPath, source)
			if err != nil {
				t.Fatalf("detect fixture vulnerability: %v", err)
			}
			if len(vulns) == 0 {
				t.Fatalf("expected fixture vulnerability")
			}
			if vulns[0].LineStart != tc.wantSinkLine || vulns[0].SinkNode.Line != tc.wantSinkLine {
				t.Fatalf("fixture sink line mismatch: got line_start=%d sink_line=%d want %d", vulns[0].LineStart, vulns[0].SinkNode.Line, tc.wantSinkLine)
			}
			export := graph.ExportReactFlow(g, &vulns[0])
			if len(export.Nodes) == 0 || len(export.Edges) == 0 {
				t.Fatalf("expected non-empty React Flow export: %+v", export)
			}
			if len(activeEdges(export)) == 0 {
				t.Fatalf("expected active fixture taint edges")
			}
			for _, ref := range vulns[0].TaintPath {
				if ref.Line <= 0 || ref.Column <= 0 || ref.ByteEnd <= ref.ByteStart {
					t.Fatalf("fixture line/byte mapping invalid: %+v", ref)
				}
			}
		})
	}
}

func TestExportReactFlowCRLFLineMapping(t *testing.T) {
	source := []byte("const id = req.query.id;\r\nconst q = \"SELECT \" + id;\r\ndb.query(q);\r\n")
	g, err := graph.NormalizeSource("crlf.js", source)
	if err != nil {
		t.Fatalf("normalize CRLF source: %v", err)
	}
	vulns, err := rules.DetectSQLInjection("crlf.js", source)
	if err != nil {
		t.Fatalf("detect CRLF SQLi: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected one CRLF SQLi finding, got %+v", vulns)
	}
	if vulns[0].SinkNode.Line != 3 {
		t.Fatalf("expected sink on absolute line 3, got %+v", vulns[0].SinkNode)
	}
	export := graph.ExportReactFlow(g, &vulns[0])
	if len(export.Nodes) == 0 {
		t.Fatalf("expected CRLF export nodes")
	}
}

func syntheticTaintGraph() (*graph.Graph, models.Vulnerability) {
	nodes := []graph.Node{
		node("n_source", graph.SemanticVariableDeclaration, "id", 0, 20, 1, 1, 1),
		node("n_taint", graph.SemanticVariableDeclaration, "query", 21, 60, 2, 1, 1),
		node("n_sink", graph.SemanticCallExpression, "db.query", 61, 76, 3, 1, 1),
		node("n_neutral", graph.SemanticIdentifier, "req", 4, 7, 1, 5, 2),
	}
	g := &graph.Graph{
		FilePath: "app.js",
		Language: "javascript",
		Nodes:    nodes,
		Edges: []graph.Edge{
			{ID: "edge:source-neutral", Source: "n_source", Target: "n_neutral", Kind: "ast_child"},
		},
	}
	v := models.Vulnerability{
		FilePath:   "app.js",
		RuleID:     "LUCID-SEC-001",
		SourceNode: astRef(nodes[0]),
		SinkNode:   astRef(nodes[2]),
		TaintPath:  []models.ASTNodeRef{astRef(nodes[0]), astRef(nodes[1]), astRef(nodes[2])},
	}
	return g, v
}

func syntheticWideGraph(width int) (*graph.Graph, models.Vulnerability) {
	g, v := syntheticTaintGraph()
	g.Nodes = append([]graph.Node(nil), g.Nodes...)
	g.Edges = append([]graph.Edge(nil), g.Edges...)
	for i := 0; i < width; i++ {
		id := "wide_" + string(rune('a'+(i%26))) + "_" + string(rune('A'+(i/26)))
		g.Nodes = append(g.Nodes, node(id, graph.SemanticIdentifier, id, uint32(100+i*5), uint32(103+i*5), 5+i, 1, 2+(i%4)))
		g.Edges = append(g.Edges, graph.Edge{ID: "edge:n_source->" + id, Source: "n_source", Target: id, Kind: "ast_child"})
	}
	return g, v
}

func node(id string, typ graph.SemanticType, name string, start, end uint32, line, col, depth int) graph.Node {
	return graph.Node{
		ID:          id,
		Type:        typ,
		CSTKind:     string(typ),
		Name:        name,
		FilePath:    "app.js",
		ByteStart:   start,
		ByteEnd:     end,
		LineStart:   line,
		ColumnStart: col,
		LineEnd:     line,
		ColumnEnd:   col + 1,
		CodeSnippet: name,
		Depth:       depth,
	}
}

func astRef(n graph.Node) models.ASTNodeRef {
	return models.ASTNodeRef{Type: string(n.Type), Name: n.Name, Line: n.LineStart, Column: n.ColumnStart, ByteStart: n.ByteStart, ByteEnd: n.ByteEnd}
}

func assertNodeType(t *testing.T, export models.GraphExport, id, want string) {
	t.Helper()
	for _, n := range export.Nodes {
		if n.ID == id {
			if n.Type != want {
				t.Fatalf("node %s type=%s want %s", id, n.Type, want)
			}
			return
		}
	}
	t.Fatalf("node %s not found in export: %+v", id, export.Nodes)
}

func assertHasNode(t *testing.T, export models.GraphExport, id string) {
	t.Helper()
	for _, n := range export.Nodes {
		if n.ID == id {
			return
		}
	}
	t.Fatalf("expected node %s", id)
}

func activeEdges(export models.GraphExport) []models.ReactFlowEdge {
	out := make([]models.ReactFlowEdge, 0)
	for _, edge := range export.Edges {
		if edge.Animated {
			out = append(out, edge)
		}
	}
	return out
}

func nodeIDs(export models.GraphExport) []string {
	ids := make([]string, 0, len(export.Nodes))
	for _, n := range export.Nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

func edgeIDs(export models.GraphExport) []string {
	ids := make([]string, 0, len(export.Edges))
	for _, e := range export.Edges {
		ids = append(ids, e.ID)
	}
	return ids
}

func nodePositions(export models.GraphExport) map[string]models.NodePosition {
	positions := make(map[string]models.NodePosition, len(export.Nodes))
	for _, n := range export.Nodes {
		positions[n.ID] = n.Position
	}
	return positions
}

func positionKey(pos models.NodePosition) string {
	return jsonNumber(pos.X) + ":" + jsonNumber(pos.Y)
}

func jsonNumber(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func nodeIDForRef(t *testing.T, g *graph.Graph, ref models.ASTNodeRef) string {
	t.Helper()
	for _, node := range g.Nodes {
		if node.ByteStart == ref.ByteStart && node.ByteEnd == ref.ByteEnd {
			return node.ID
		}
	}
	t.Fatalf("node not found for ref %+v", ref)
	return ""
}
