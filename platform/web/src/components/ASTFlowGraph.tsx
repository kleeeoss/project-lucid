// src/components/ASTFlowGraph.tsx
"use client";

import React, { useMemo, useState } from "react";
import {
    ReactFlow,
    Background,
    Controls,
    MiniMap,
    Node,
    Edge,
    Position,
    Handle,
    useReactFlow,
    ReactFlowProvider,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { AlertCircle, ArrowRight, Sparkles, CheckCircle2 } from "lucide-react";
import { Vulnerability } from "@/lib/types";

interface NodeData {
    type: "source" | "propagator" | "sink" | "sanitizer";
    label: string;
    line: number;
    codeSnippet?: string;
}

interface LocalASTNode {
    id: string;
    type: "source" | "propagator" | "sink" | "sanitizer";
    label: string;
    line: number;
    column?: number;
    codeSnippet?: string;
}

interface LocalASTEdge {
    id: string;
    source: string;
    target: string;
    label?: string;
    animated?: boolean;
}

// Cyber-Industrial Custom Node
function CustomNode({ data }: { data: NodeData }) {
    const configs: Record<string, { border: string; bg: string; text: string; icon: React.ReactNode }> = {
        source: {
            border: "border-emerald-500 shadow-emerald-500/10",
            bg: "bg-emerald-950/40",
            text: "text-emerald-400",
            icon: <Sparkles className="h-3 w-3" />,
        },
        propagator: {
            border: "border-amber-500 shadow-amber-500/10",
            bg: "bg-amber-950/40",
            text: "text-amber-400",
            icon: <ArrowRight className="h-3 w-3" />,
        },
        sink: {
            border: "border-rose-500 shadow-rose-500/20 ring-1 ring-rose-500/40",
            bg: "bg-rose-950/40",
            text: "text-rose-400",
            icon: <AlertCircle className="h-3 w-3" />,
        },
        sanitizer: {
            border: "border-sky-500 shadow-sky-500/10",
            bg: "bg-sky-950/40",
            text: "text-sky-400",
            icon: <CheckCircle2 className="h-3 w-3" />,
        },
    };

    const style = configs[data.type] || configs.propagator;

    return (
        <div
            className={`w-72 rounded-xl border-2 p-3.5 shadow-2xl backdrop-blur-md transition-all bg-zinc-950/90 ${style.border}`}
        >
            <Handle type="target" position={Position.Top} className="!bg-zinc-500 !w-2 !h-2" />
            <div className="flex items-center justify-between text-[10px] uppercase font-bold tracking-wider">
                <span className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full ${style.bg} ${style.text}`}>
                    {style.icon}
                    {data.type}
                </span>
                <span className="font-mono text-zinc-500">L{data.line}</span>
            </div>

            <div className="mt-2 text-xs font-bold text-white font-mono truncate">{data.label}</div>

            {data.codeSnippet && (
                <pre className="mt-2 rounded-lg bg-black/80 border border-zinc-800/80 p-2 font-mono text-[11px] text-zinc-300 overflow-hidden text-ellipsis whitespace-nowrap">
                    <code>{data.codeSnippet}</code>
                </pre>
            )}
            <Handle type="source" position={Position.Bottom} className="!bg-zinc-500 !w-2 !h-2" />
        </div>
    );
}

// Inner Canvas with Camera Stepper Control
function FlowCanvas({ vuln }: { vuln: Vulnerability }) {
    const { setCenter } = useReactFlow();
    const [currentStep, setCurrentStep] = useState(0);

    const nodeTypes = useMemo(() => ({ custom: CustomNode }), []);

    // Compute Layout with explicit typing
    const { nodes, edges } = useMemo<{ nodes: Node[]; edges: Edge[] }>(() => {
        const rawGraph = vuln.ast_graph_json as { nodes?: LocalASTNode[]; edges?: LocalASTEdge[] } | null | undefined;

        if (rawGraph && rawGraph.nodes && rawGraph.nodes.length > 0) {
            const autoNodes: Node[] = rawGraph.nodes.map((n: LocalASTNode, idx: number) => ({
                id: n.id,
                type: "custom",
                position: { x: 200, y: idx * 140 + 30 },
                data: {
                    type: n.type,
                    label: n.label,
                    line: n.line,
                    codeSnippet: n.codeSnippet,
                },
            }));

            const autoEdges: Edge[] = (rawGraph.edges || []).map((e: LocalASTEdge) => ({
                id: e.id,
                source: e.source,
                target: e.target,
                animated: e.animated ?? true,
                style: { stroke: "#f59e0b", strokeWidth: 2 },
                label: e.label,
                labelStyle: { fill: "#f59e0b", fontSize: 10, fontFamily: "monospace" },
            }));

            return { nodes: autoNodes, edges: autoEdges };
        }

        // Default 3-node fallback layout
        const startLine = vuln.line_start || 42;
        const fallbackNodes: Node[] = [
            {
                id: "source",
                type: "custom",
                position: { x: 200, y: 30 },
                data: {
                    type: "source",
                    label: "req.params.id (HTTP Input)",
                    line: startLine,
                    codeSnippet: "const userId = req.params.id;",
                },
            },
            {
                id: "propagator",
                type: "custom",
                position: { x: 200, y: 170 },
                data: {
                    type: "propagator",
                    label: "Template Literal String Interpolation",
                    line: startLine + 1,
                    codeSnippet: "query = `SELECT * FROM users WHERE id = '${userId}'`",
                },
            },
            {
                id: "sink",
                type: "custom",
                position: { x: 200, y: 310 },
                data: {
                    type: "sink",
                    label: "db.query() (Raw SQL Execution Sink)",
                    line: startLine + 2,
                    codeSnippet: "await db.query(query);",
                },
            },
        ];

        const fallbackEdges: Edge[] = [
            {
                id: "e1",
                source: "source",
                target: "propagator",
                animated: true,
                style: { stroke: "#f59e0b", strokeWidth: 2 },
                label: "taint propagated",
                labelStyle: { fill: "#f59e0b", fontSize: 10, fontFamily: "monospace" },
            },
            {
                id: "e2",
                source: "propagator",
                target: "sink",
                animated: true,
                style: { stroke: "#f43f5e", strokeWidth: 2.5 },
                label: "untrusted execution",
                labelStyle: { fill: "#f43f5e", fontSize: 10, fontFamily: "monospace" },
            },
        ];

        return { nodes: fallbackNodes, edges: fallbackEdges };
    }, [vuln]);

    // Stepper Controller: Focus camera on node step
    const handleStep = (stepIdx: number) => {
        if (!nodes[stepIdx]) return;
        setCurrentStep(stepIdx);
        const targetNode = nodes[stepIdx];
        setCenter(targetNode.position.x + 140, targetNode.position.y + 50, {
            zoom: 1.15,
            duration: 600,
        });
    };

    return (
        <div className="relative h-[480px] w-full rounded-xl border border-zinc-800 bg-[#090b10] overflow-hidden">
            {/* Top Floating Taint Execution Stepper */}
            <div className="absolute top-3 left-3 z-10 flex items-center gap-1.5 bg-zinc-950/80 backdrop-blur-md border border-zinc-800 p-1.5 rounded-lg text-xs font-mono">
                <span className="px-2 text-zinc-400 font-semibold uppercase text-[10px]">Taint Path:</span>
                {nodes.map((n: Node, idx: number) => (
                    <button
                        key={n.id}
                        onClick={() => handleStep(idx)}
                        className={`px-2 py-0.5 rounded text-[11px] font-medium transition ${currentStep === idx
                            ? "bg-zinc-700 text-white shadow-sm border border-zinc-600"
                            : "text-zinc-500 hover:text-zinc-300"
                            }`}
                    >
                        Step {idx + 1}
                    </button>
                ))}
            </div>

            <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} fitView attributionPosition="bottom-right">
                <Background color="#1e2433" gap={20} size={1} />
                <Controls className="!bg-zinc-900 !border-zinc-800 !text-white" />
                <MiniMap
                    nodeColor={(n: Node) => {
                        const nodeData = n.data as unknown as NodeData | undefined;
                        if (nodeData?.type === "source") return "#10b981";
                        if (nodeData?.type === "sink") return "#f43f5e";
                        return "#f59e0b";
                    }}
                    maskColor="rgba(9, 11, 16, 0.85)"
                    className="!bg-zinc-950 !border !border-zinc-800 rounded-lg overflow-hidden"
                />
            </ReactFlow>
        </div>
    );
}

export function ASTFlowGraph({ vuln }: { vuln: Vulnerability }) {
    return (
        <ReactFlowProvider>
            <FlowCanvas vuln={vuln} />
        </ReactFlowProvider>
    );
}