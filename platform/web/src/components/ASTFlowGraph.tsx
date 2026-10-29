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
import { AlertCircle, ArrowRight, Sparkles, CheckCircle2, Network } from "lucide-react";
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

// Impeccable DevSecOps AST Node Card
function CustomNode({ data }: { data: NodeData }) {
    const configs: Record<string, { border: string; bg: string; text: string; icon: React.ReactNode }> = {
        source: {
            border: "border-amber-500/40 hover:border-amber-500/80 shadow-amber-500/5",
            bg: "bg-amber-500/10",
            text: "text-amber-400 border border-amber-500/20",
            icon: <Sparkles className="h-3 w-3" />,
        },
        propagator: {
            border: "border-indigo-500/40 hover:border-indigo-500/80 shadow-indigo-500/5",
            bg: "bg-indigo-500/10",
            text: "text-indigo-400 border border-indigo-500/20",
            icon: <ArrowRight className="h-3 w-3" />,
        },
        sink: {
            border: "border-rose-500/60 hover:border-rose-500/90 shadow-rose-500/10 ring-1 ring-rose-500/40",
            bg: "bg-rose-500/10",
            text: "text-rose-400 border border-rose-500/20",
            icon: <AlertCircle className="h-3 w-3" />,
        },
        sanitizer: {
            border: "border-emerald-500/40 hover:border-emerald-500/80 shadow-emerald-500/5",
            bg: "bg-emerald-500/10",
            text: "text-emerald-400 border border-emerald-500/20",
            icon: <CheckCircle2 className="h-3 w-3" />,
        },
    };

    const style = configs[data.type] || configs.propagator;

    return (
        <div
            className={`w-72 rounded-lg border p-3 shadow-xl backdrop-blur-md transition-all bg-zinc-950/95 ${style.border}`}
        >
            <Handle type="target" position={Position.Top} className="!bg-zinc-600 !w-2 !h-2 !border-none" />
            <div className="flex items-center justify-between text-[10px] uppercase font-mono font-semibold tracking-wider">
                <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded ${style.bg} ${style.text}`}>
                    {style.icon}
                    {data.type}
                </span>
                <span className="font-mono text-zinc-400 bg-zinc-900 border border-zinc-800 px-1.5 py-0.5 rounded text-[10px]">
                    L{data.line}
                </span>
            </div>

            <div className="mt-2 text-xs font-semibold text-zinc-100 font-mono truncate">{data.label}</div>

            {data.codeSnippet && (
                <pre className="mt-2 rounded bg-black/90 border border-zinc-800/80 p-2 font-mono text-[11px] text-zinc-300 overflow-hidden text-ellipsis whitespace-nowrap">
                    <code>{data.codeSnippet}</code>
                </pre>
            )}
            <Handle type="source" position={Position.Bottom} className="!bg-zinc-600 !w-2 !h-2 !border-none" />
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

            // Smooth animated red edges (#ef4444) for security taint propagation
            const autoEdges: Edge[] = (rawGraph.edges || []).map((e: LocalASTEdge) => ({
                id: e.id,
                source: e.source,
                target: e.target,
                animated: true,
                style: { stroke: "#ef4444", strokeWidth: 2 },
                label: e.label,
                labelStyle: { fill: "#ef4444", fontSize: 10, fontFamily: "monospace" },
            }));

            return { nodes: autoNodes, edges: autoEdges };
        }

        // Truthful empty layout when no structured AST graph exists
        return { nodes: [], edges: [] };
    }, [vuln]);

    // Stepper Controller: Focus camera on node step
    const handleStep = (stepIdx: number) => {
        if (!nodes[stepIdx]) return;
        setCurrentStep(stepIdx);
        const targetNode = nodes[stepIdx];
        setCenter(targetNode.position.x + 140, targetNode.position.y + 50, {
            zoom: 1.15,
            duration: 500,
        });
    };

    if (nodes.length === 0) {
        return (
            <div className="flex flex-col items-center justify-center h-[440px] w-full rounded-lg border border-zinc-800 bg-[#09090b] p-6 text-center">
                <Network className="h-8 w-8 text-zinc-600 mb-2" />
                <p className="text-sm font-semibold text-zinc-200">AST Taint Trace Unavailable</p>
                <p className="text-xs text-zinc-500 mt-1 max-w-sm">
                    Static engine flagged this finding via pattern rules; no structured AST taint dataflow graph was recorded.
                </p>
            </div>
        );
    }

    return (
        <div className="relative h-[480px] w-full rounded-lg border border-zinc-800 bg-[#09090b] overflow-hidden">
            {/* Top Floating Taint Execution Stepper */}
            <div className="absolute top-3 left-3 z-10 flex items-center gap-1.5 bg-zinc-950/90 backdrop-blur-md border border-zinc-800 p-1.5 rounded-md text-xs font-mono shadow-md">
                <span className="px-2 text-zinc-400 font-semibold uppercase text-[10px]">Taint Path:</span>
                {nodes.map((n: Node, idx: number) => (
                    <button
                        key={n.id}
                        onClick={() => handleStep(idx)}
                        className={`px-2 py-0.5 rounded text-[11px] font-medium transition ${
                            currentStep === idx
                                ? "bg-zinc-800 text-white shadow-sm border border-zinc-700"
                                : "text-zinc-500 hover:text-zinc-300"
                        }`}
                    >
                        Step {idx + 1}
                    </button>
                ))}
            </div>

            <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} fitView attributionPosition="bottom-right">
                <Background color="#18181b" gap={20} size={1} />
                <Controls className="!bg-zinc-900 !border-zinc-800 !text-zinc-300 rounded-md overflow-hidden" />
                <MiniMap
                    nodeColor={(n: Node) => {
                        const nodeData = n.data as unknown as NodeData | undefined;
                        if (nodeData?.type === "source") return "#f59e0b";
                        if (nodeData?.type === "sink") return "#ef4444";
                        if (nodeData?.type === "sanitizer") return "#10b981";
                        return "#6366f1";
                    }}
                    maskColor="rgba(9, 9, 11, 0.85)"
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
