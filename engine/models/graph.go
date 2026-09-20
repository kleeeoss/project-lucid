package models

type GraphExport struct {
	Nodes []ReactFlowNode `json:"nodes"`
	Edges []ReactFlowEdge `json:"edges"`
}

type ReactFlowNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Position NodePosition   `json:"position"`
	Data     map[string]any `json:"data"`
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ReactFlowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Label    string `json:"label,omitempty"`
	Animated bool   `json:"animated"`
	Style    string `json:"style,omitempty"`
}
