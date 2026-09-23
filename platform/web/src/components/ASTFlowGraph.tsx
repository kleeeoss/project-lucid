"use client";

import React, { useMemo } from "react";
import {
    ReactFlow,
    Background,
    Controls,
    MiniMap,
    Node,
    Edge,
    Position,
    Handle,
} from "@xyflow/react";

// Custom Node component for AST Taint Flow
function CustomNode({ data }: { data: any }) {
    const borderStyles: Record<string, string> = {
        source: "border-emerald-500 bg-emerald-950/40 text-emerald-300 shadow-emerald-500/20",
        propagator: "border-amber-500 bg-amber-950/40 text-amber-300 shadow-amber-500/20",
        sink: "border-rose-500 bg-rose-950/40 text-rose-300 shadow-rose-500/20",
    };

    const badgeColors: Record<string, string> = {
        source: "bg-emerald-500/20 text-emerald-400",
        propagator: "bg-amber-500/20 text-amber-400",
        sink: "bg-rose-500/20 text-rose-400",
    };

    return (
        <div className={`w-64 rounded-lg border-2 p-3 shadow-lg backdrop-blur-md transition-all ${borderStyles[data.type] || borderStyles.propagator}`}>
            <Handle type="target" position={Position.Top} className="!bg-slate-400" />
            <div className="flex items-center justify-between text-[10px] uppercase font-bold tracking-wider">
                <span className={`rounded px-1.5 py-0.5 ${badgeColors[data.type]}`}>{data.type}</span>
                <span className="text-slate-400">Line {data.line}</span>
            </div>
            <div className="mt-1.5 text-sm font-semibold truncate text-white">{data.label}</div>
            {data.codeSnippet && (
                <code className="mt-2 block rounded bg-slate-900/90 p-1.5 font-mono text-[11px] text-slate-300 truncate">
                    {data.codeSnippet}
                </code>
            )}
            <Handle type="source" position={Position.Bottom} className="!bg-slate-400" />
        </div>
    );
}

export function ASTFlowGraph({ vuln }: { vuln: any }) {
    const nodeTypes = useMemo(() => ({ custom: CustomNode }), []);

    // Construct initial nodes & edges from vulnerability taint trace
    const { nodes, edges } = useMemo(() => {
        const rawNodes: Node[] = [
            {
                id: "source-1",
                type: "custom",
                position: { x: 150, y: 30 },
                data: {
                    type: "source",
                    label: "req.query.id",
                    line: vuln.line_start || 42,
                    codeSnippet: "const id = req.query.id;",
                },
            },
            {
                id: "prop-1",
                type: "custom",
                position: { x: 150, y: 170 },
                data: {
                    type: "propagator",
                    label: "String Concatenation",
                    line: (vuln.line_start || 42) + 1,
                    codeSnippet: "sql = 'SELECT * FROM users...' + id",
                },
            },
            {
                id: "sink-1",
                type: "custom",
                position: { x: 150, y: 310 },
                data: {
                    type: "sink",
                    label: "db.query()",
                    line: (vuln.line_start || 42) + 2,
                    codeSnippet: "await db.query(sql);",
                },
            },
        ];

        const rawEdges: Edge[] = [
            {
                id: "e1-2",
                source: "source-1",
                target: "prop-1",
                animated: true,
                style: { stroke: "#f59e0b", strokeWidth: 2 },
                label: "taint propagation",
                labelStyle: { fill: "#94a3b8", fontSize: 10 },
            },
            {
                id: "e2-3",
                source: "prop-1",
                target: "sink-1",
                animated: true,
                style: { stroke: "#ef4444", strokeWidth: 2 },
                label: "untrusted execution",
                labelStyle: { fill: "#ef4444", fontSize: 10 },
            },
        ];

        return { nodes: rawNodes, edges: rawEdges };
    }, [vuln]);

    return (
        <div className="h-[420px] w-full rounded-xl border border-slate-800 bg-[#0a0f1d] overflow-hidden">
            <ReactFlow
                nodes={nodes}
                edges={edges}
                nodeTypes={nodeTypes}
                fitView
                attributionPosition="bottom-right"
            >
                <Background color="#1e293b" gap={16} size={1} />
                <Controls />
                <MiniMap
                    nodeColor={(n: any) => {
                        if (n.data?.type === "source") return "#10b981";
                        if (n.data?.type === "sink") return "#ef4444";
                        return "#f59e0b";
                    }}
                    maskColor="rgba(10, 15, 29, 0.8)"
                    style={{ backgroundColor: "#0f172a" }}
                />
            </ReactFlow>
        </div>
    );
}