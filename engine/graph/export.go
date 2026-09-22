package graph

import (
	"fmt"
	"sort"

	"lucid-ci/engine/models"
)

const (
	ReactFlowSourceNode = "sourceNode"
	ReactFlowTaintNode  = "taintNode"
	ReactFlowSinkNode   = "sinkNode"
	ReactFlowASTNode    = "astNode"

	activeEdgeStyle     = "stroke: #ef4444; stroke-width: 2.5px;"
	structuralEdgeStyle = "stroke: #94a3b8; stroke-width: 1px;"

	maxReactFlowNodes = 50
	layoutXSpacing    = 220.0
	layoutYSpacing    = 100.0
)

func ExportReactFlow(g *Graph, v *models.Vulnerability) models.GraphExport {
	if g == nil || len(g.Nodes) == 0 {
		return models.GraphExport{Nodes: []models.ReactFlowNode{}, Edges: []models.ReactFlowEdge{}}
	}

	orderedNodes := sortedNodes(g.Nodes)
	nodeByID := make(map[string]Node, len(orderedNodes))
	for _, node := range orderedNodes {
		nodeByID[node.ID] = node
	}

	pathIDs := matchTaintPath(orderedNodes, v)
	sourceNodeRef, hasSource := sourceRef(v)
	sinkNodeRef, hasSink := sinkRef(v)
	sourceID := matchSingleRef(orderedNodes, sourceNodeRef, hasSource)
	sinkID := matchSingleRef(orderedNodes, sinkNodeRef, hasSink)
	selectedIDs := selectExportNodeIDs(orderedNodes, g.Edges, pathIDs, sourceID, sinkID)
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}

	pathSet := make(map[string]bool, len(pathIDs))
	for _, id := range pathIDs {
		pathSet[id] = true
	}
	activePairs := consecutivePairs(pathIDs)
	positions := layoutPositions(selectedIDs, nodeByID, pathIDs)

	flowNodes := make([]models.ReactFlowNode, 0, len(selectedIDs))
	for _, id := range selectedIDs {
		node, ok := nodeByID[id]
		if !ok {
			continue
		}
		flowNodes = append(flowNodes, models.ReactFlowNode{
			ID:       node.ID,
			Type:     reactFlowNodeType(node.ID, sourceID, sinkID, pathSet),
			Position: positions[node.ID],
			Data: map[string]any{
				"label":         nodeLabel(node),
				"nodeType":      string(node.Type),
				"codeSnippet":   node.CodeSnippet,
				"isTainted":     pathSet[node.ID] || node.ID == sourceID || node.ID == sinkID,
				"line":          node.LineStart,
				"col":           node.ColumnStart,
				"filePath":      node.FilePath,
				"byteStart":     node.ByteStart,
				"byteEnd":       node.ByteEnd,
				"containsError": node.ContainsError,
			},
		})
	}

	flowEdges := exportEdges(g.Edges, selected, activePairs)
	flowEdges = appendMissingActiveTaintEdges(flowEdges, selected, activePairs)
	sort.SliceStable(flowEdges, func(i, j int) bool { return flowEdges[i].ID < flowEdges[j].ID })

	return models.GraphExport{Nodes: flowNodes, Edges: flowEdges}
}

func sortedNodes(nodes []Node) []Node {
	out := append([]Node(nil), nodes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ByteStart != out[j].ByteStart {
			return out[i].ByteStart < out[j].ByteStart
		}
		if out[i].ByteEnd != out[j].ByteEnd {
			return out[i].ByteEnd < out[j].ByteEnd
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sourceRef(v *models.Vulnerability) (models.ASTNodeRef, bool) {
	if v == nil {
		return models.ASTNodeRef{}, false
	}
	return v.SourceNode, true
}

func sinkRef(v *models.Vulnerability) (models.ASTNodeRef, bool) {
	if v == nil {
		return models.ASTNodeRef{}, false
	}
	return v.SinkNode, true
}

func matchTaintPath(nodes []Node, v *models.Vulnerability) []string {
	if v == nil || len(v.TaintPath) == 0 {
		return nil
	}
	ids := make([]string, 0, len(v.TaintPath))
	seen := map[string]bool{}
	for _, ref := range v.TaintPath {
		id := matchRef(nodes, ref)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func matchSingleRef(nodes []Node, ref models.ASTNodeRef, ok bool) string {
	if !ok {
		return ""
	}
	return matchRef(nodes, ref)
}

func matchRef(nodes []Node, ref models.ASTNodeRef) string {
	for _, node := range nodes {
		if node.ByteStart == ref.ByteStart && node.ByteEnd == ref.ByteEnd && string(node.Type) == ref.Type {
			return node.ID
		}
	}
	for _, node := range nodes {
		if node.ByteStart == ref.ByteStart && node.ByteEnd == ref.ByteEnd {
			return node.ID
		}
	}
	for _, node := range nodes {
		if node.LineStart == ref.Line && node.ColumnStart == ref.Column && node.Name == ref.Name {
			return node.ID
		}
	}
	for _, node := range nodes {
		if node.LineStart == ref.Line && node.ColumnStart == ref.Column {
			return node.ID
		}
	}
	return ""
}

func selectExportNodeIDs(nodes []Node, edges []Edge, pathIDs []string, sourceID, sinkID string) []string {
	selected := map[string]bool{}
	ordered := make([]string, 0, maxReactFlowNodes)
	add := func(id string) {
		if id == "" || selected[id] || len(ordered) >= maxReactFlowNodes {
			return
		}
		selected[id] = true
		ordered = append(ordered, id)
	}

	add(sourceID)
	for _, id := range pathIDs {
		add(id)
	}
	add(sinkID)

	pathSet := map[string]bool{}
	for _, id := range pathIDs {
		pathSet[id] = true
	}
	if sourceID != "" {
		pathSet[sourceID] = true
	}
	if sinkID != "" {
		pathSet[sinkID] = true
	}

	for _, edge := range sortedEdges(edges) {
		if pathSet[edge.Source] {
			add(edge.Target)
		}
		if pathSet[edge.Target] {
			add(edge.Source)
		}
	}
	for _, node := range nodes {
		add(node.ID)
	}
	return ordered
}

func sortedEdges(edges []Edge) []Edge {
	out := append([]Edge(nil), edges...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func consecutivePairs(pathIDs []string) map[string]bool {
	pairs := make(map[string]bool)
	for i := 0; i+1 < len(pathIDs); i++ {
		if pathIDs[i] == "" || pathIDs[i+1] == "" || pathIDs[i] == pathIDs[i+1] {
			continue
		}
		pairs[edgePairKey(pathIDs[i], pathIDs[i+1])] = true
	}
	return pairs
}

func edgePairKey(source, target string) string { return source + "\x00" + target }

func layoutPositions(selectedIDs []string, nodeByID map[string]Node, pathIDs []string) map[string]models.NodePosition {
	positions := make(map[string]models.NodePosition, len(selectedIDs))
	used := map[string]bool{}
	pathIndex := map[string]int{}
	for i, id := range pathIDs {
		if _, ok := nodeByID[id]; ok {
			pathIndex[id] = i
		}
	}

	place := func(id string, x, y float64) {
		for used[positionKey(x, y)] {
			y += layoutYSpacing
		}
		positions[id] = models.NodePosition{X: x, Y: y}
		used[positionKey(x, y)] = true
	}

	for _, id := range selectedIDs {
		if idx, ok := pathIndex[id]; ok {
			place(id, float64(idx)*layoutXSpacing, 0)
		}
	}
	for i, id := range selectedIDs {
		if _, ok := positions[id]; ok {
			continue
		}
		node := nodeByID[id]
		level := node.Depth
		if level < 1 {
			level = 1
		}
		place(id, float64(i%maxReactFlowNodes)*layoutXSpacing, float64(level)*layoutYSpacing)
	}
	return positions
}

func positionKey(x, y float64) string { return fmt.Sprintf("%.2f:%.2f", x, y) }

func reactFlowNodeType(id, sourceID, sinkID string, pathSet map[string]bool) string {
	if id == sourceID {
		return ReactFlowSourceNode
	}
	if id == sinkID {
		return ReactFlowSinkNode
	}
	if pathSet[id] {
		return ReactFlowTaintNode
	}
	return ReactFlowASTNode
}

func nodeLabel(node Node) string {
	if node.Name != "" {
		return node.Name
	}
	if node.CodeSnippet != "" {
		return node.CodeSnippet
	}
	return string(node.Type)
}

func exportEdges(edges []Edge, selected map[string]bool, activePairs map[string]bool) []models.ReactFlowEdge {
	seen := map[string]bool{}
	out := make([]models.ReactFlowEdge, 0, len(edges))
	for _, edge := range sortedEdges(edges) {
		if !selected[edge.Source] || !selected[edge.Target] {
			continue
		}
		id := edge.ID
		if id == "" {
			id = fmt.Sprintf("edge:%s->%s", edge.Source, edge.Target)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		active := activePairs[edgePairKey(edge.Source, edge.Target)]
		out = append(out, models.ReactFlowEdge{
			ID:       id,
			Source:   edge.Source,
			Target:   edge.Target,
			Label:    edge.Kind,
			Animated: active,
			Style:    edgeStyle(active),
		})
	}
	return out
}

func appendMissingActiveTaintEdges(edges []models.ReactFlowEdge, selected map[string]bool, activePairs map[string]bool) []models.ReactFlowEdge {
	existingPairs := map[string]bool{}
	for _, edge := range edges {
		existingPairs[edgePairKey(edge.Source, edge.Target)] = true
	}
	keys := make([]string, 0, len(activePairs))
	for key := range activePairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		source, target := splitPairKey(key)
		if !selected[source] || !selected[target] || existingPairs[key] {
			continue
		}
		edges = append(edges, models.ReactFlowEdge{
			ID:       fmt.Sprintf("edge:taint:%s->%s", source, target),
			Source:   source,
			Target:   target,
			Label:    "taint_flow",
			Animated: true,
			Style:    activeEdgeStyle,
		})
	}
	return edges
}

func splitPairKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func edgeStyle(active bool) string {
	if active {
		return activeEdgeStyle
	}
	return structuralEdgeStyle
}
