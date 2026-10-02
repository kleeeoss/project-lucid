package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphExportJSONContract(t *testing.T) {
	export := GraphExport{
		Nodes: []ReactFlowNode{{
			ID:       "node-1",
			Type:     "sourceNode",
			Position: NodePosition{X: 10, Y: 20},
			Data:     map[string]any{"label": "req.query.id", "nodeType": "Identifier", "isTainted": true},
		}},
		Edges: []ReactFlowEdge{{ID: "edge-1", Source: "node-1", Target: "node-2", Animated: true, Style: "stroke: #ef4444"}},
	}
	b, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal graph export: %v", err)
	}
	text := string(b)
	for _, required := range []string{"\"nodes\"", "\"edges\"", "\"position\"", "\"data\"", "\"animated\""} {
		if !strings.Contains(text, required) {
			t.Fatalf("expected JSON to contain %s: %s", required, text)
		}
	}
}
