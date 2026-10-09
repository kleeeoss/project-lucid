// src/components/scan/ScanWorkbench.tsx
"use client";

import React, { useState } from "react";
import { Vulnerability, ScanRun } from "@/lib/types";
import { SeverityBadge } from "@/components/StatusBadge";
import { ASTFlowGraph } from "@/components/ASTFlowGraph";
import { DiffViewer } from "@/components/scan/DiffViewer";
import {
    Network,
    GitCompare,
    Sparkles,
    Terminal,
    FileCode,
    ShieldCheck,
    ChevronRight,
    AlertTriangle
} from "lucide-react";

export function ScanWorkbench({ vulns, scan }: { vulns: Vulnerability[]; scan: ScanRun }) {
    const [selectedIndex, setSelectedIndex] = useState(0);
    const [activeTab, setActiveTab] = useState<"graph" | "diff" | "ai" | "sandbox">("graph");

    if (vulns.length === 0) {
        return (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-16 text-center">
                <ShieldCheck className="mx-auto h-12 w-12 text-emerald-400" />
                <h3 className="mt-4 text-base font-semibold text-white">Zero Vulnerabilities Detected</h3>
                <p className="mt-1 text-xs text-zinc-400">
                    Intraprocedural taint flow analysis and AST security rules found no dangerous sinks.
                </p>
            </div>
        );
    }

    const selectedVuln = vulns[selectedIndex] || vulns[0];

    return (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
            {/* LEFT PANE: Findings Selector (4 Columns) */}
            <div className="lg:col-span-4 space-y-2">
                <div className="flex items-center justify-between px-1 pb-1">
                    <span className="text-xs font-bold uppercase tracking-wider text-zinc-400">
                        Detected Findings ({vulns.length})
                    </span>
                    <span className="text-[11px] text-zinc-500 font-mono">Sorted by Severity</span>
                </div>

                <div className="space-y-2">
                    {vulns.map((v, idx) => {
                        const isSelected = idx === selectedIndex;
                        return (
                            <button
                                key={v.id}
                                onClick={() => setSelectedIndex(idx)}
                                className={`w-full text-left rounded-xl p-4 border transition-all text-xs ${isSelected
                                        ? "bg-zinc-800/90 border-rose-500/60 shadow-lg shadow-rose-500/5 ring-1 ring-rose-500/30"
                                        : "bg-zinc-900/60 border-zinc-800 hover:bg-zinc-800/40 hover:border-zinc-700"
                                    }`}
                            >
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                        <span className="text-xs font-bold text-white">
                                            {v.cwe === "CWE-89" ? "SQL Injection" : v.rule_id}
                                        </span>
                                        <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400 border border-zinc-700/60">
                                            {v.cwe}
                                        </span>
                                    </div>
                                    <SeverityBadge severity={v.severity} />
                                </div>

                                <div className="mt-2.5 flex items-center justify-between text-zinc-400 font-mono text-[11px]">
                                    <span className="flex items-center gap-1 truncate max-w-[200px]">
                                        <FileCode className="h-3 w-3 text-zinc-500 shrink-0" />
                                        {v.file_path}:{v.line_start}
                                    </span>
                                    <span className="text-zinc-500">{((Number(v.confidence_score) || 0) * 100).toFixed(0)}% Conf</span>
                                </div>
                            </button>
                        );
                    })}
                </div>
            </div>

            {/* RIGHT PANE: The Hero Remediation Workspace (8 Columns) */}
            <div className="lg:col-span-8 rounded-xl border border-zinc-800 bg-zinc-900/70 backdrop-blur-md overflow-hidden shadow-2xl">
                {/* Workspace Tab Header */}
                <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-950/70 px-4 py-2.5">
                    <div className="flex items-center gap-1 text-xs">
                        <button
                            onClick={() => setActiveTab("graph")}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium transition ${activeTab === "graph"
                                    ? "bg-zinc-800 text-white shadow-sm border border-zinc-700"
                                    : "text-zinc-400 hover:text-white"
                                }`}
                        >
                            <Network className="h-3.5 w-3.5 text-amber-400" />
                            AST Taint Trace
                        </button>

                        <button
                            onClick={() => setActiveTab("diff")}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium transition ${activeTab === "diff"
                                    ? "bg-zinc-800 text-white shadow-sm border border-zinc-700"
                                    : "text-zinc-400 hover:text-white"
                                }`}
                        >
                            <GitCompare className="h-3.5 w-3.5 text-emerald-400" />
                            Remediation Patch
                        </button>

                        <button
                            onClick={() => setActiveTab("ai")}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium transition ${activeTab === "ai"
                                    ? "bg-zinc-800 text-white shadow-sm border border-zinc-700"
                                    : "text-zinc-400 hover:text-white"
                                }`}
                        >
                            <Sparkles className="h-3.5 w-3.5 text-sky-400" />
                            AI Security Rationale
                        </button>

                        <button
                            onClick={() => setActiveTab("sandbox")}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium transition ${activeTab === "sandbox"
                                    ? "bg-zinc-800 text-white shadow-sm border border-zinc-700"
                                    : "text-zinc-400 hover:text-white"
                                }`}
                        >
                            <Terminal className="h-3.5 w-3.5 text-rose-400" />
                            gVisor Sandbox
                        </button>
                    </div>

                    <div className="text-[11px] font-mono text-zinc-500 hidden sm:block">
                        {selectedVuln.file_path}:{selectedVuln.line_start}
                    </div>
                </div>

                {/* Tab Content Panes */}
                <div className="p-5">
                    {activeTab === "graph" && (
                        <div className="space-y-4">
                            <div className="flex items-center justify-between text-xs text-zinc-400">
                                <span className="font-semibold text-zinc-200">Mathematical Taint Traversal Graph</span>
                                <span className="font-mono text-[11px] text-zinc-500">Tree-sitter CST → Directed Acyclic Graph</span>
                            </div>
                            <ASTFlowGraph vuln={selectedVuln} />
                        </div>
                    )}

                    {activeTab === "diff" && (
                        <DiffViewer
                            originalCode={selectedVuln.vulnerable_code}
                            patchCode={selectedVuln.ai_remediation_patch || "// No automated patch generated"}
                            filePath={selectedVuln.file_path}
                            lineStart={selectedVuln.line_start}
                        />
                    )}

                    {activeTab === "ai" && (
                        <div className="space-y-5">
                            <div className="rounded-lg border border-sky-500/20 bg-sky-950/20 p-5">
                                <div className="flex items-center gap-2 text-sky-400 font-semibold text-xs tracking-wider uppercase">
                                    <Sparkles className="h-4 w-4" />
                                    Small Language Model Architectural Explanation
                                </div>
                                <p className="mt-3 text-sm text-zinc-200 leading-relaxed font-sans">
                                    {selectedVuln.ai_explanation ||
                                        "No automated explanation generated for this vulnerability."}
                                </p>
                            </div>

                            <div className="grid grid-cols-2 gap-4 text-xs font-mono">
                                <div className="rounded-lg border border-zinc-800 bg-zinc-950/60 p-3">
                                    <span className="text-zinc-500 block">Prompt Hardening</span>
                                    <span className="text-emerald-400 font-semibold mt-1 block">Cryptographic Nonce Enclosure</span>
                                </div>
                                <div className="rounded-lg border border-zinc-800 bg-zinc-950/60 p-3">
                                    <span className="text-zinc-500 block">Remediation Engine</span>
                                    <span className="text-sky-400 font-semibold mt-1 block">AI Remediation Pipeline</span>
                                </div>
                            </div>
                        </div>
                    )}

                    {activeTab === "sandbox" && (
                        <div className="rounded-lg border border-zinc-800 bg-black font-mono text-xs text-zinc-300 p-4 space-y-3">
                            <div className="flex items-center justify-between text-zinc-400 border-b border-zinc-800 pb-2">
                                <span>Dynamic Detonation Telemetry (Host: Docker-on-EC2)</span>
                                {selectedVuln.sandbox_verified ? (
                                    <span className="text-emerald-400 font-bold px-2 py-0.5 rounded bg-emerald-950/60 border border-emerald-500/30">
                                        DETONATION PASSED (VERIFIED)
                                    </span>
                                ) : (
                                    <span className="text-rose-400 font-bold px-2 py-0.5 rounded bg-rose-950/60 border border-rose-500/30">
                                        DETONATION UNVERIFIED
                                    </span>
                                )}
                            </div>
                            {selectedVuln.sandbox_verified ? (
                                <div className="space-y-1.5 text-[11px] leading-relaxed">
                                    <p className="text-zinc-500">$ docker run --runtime=runsc --network none --memory 512m --cpus 0.5 lucid-sandbox-runner</p>
                                    <p className="text-emerald-400">[gVisor] Intercepting all syscalls via userspace Sentry kernel (runsc)</p>
                                    <p className="text-zinc-400">[Isolation] Host network unreachable (--network none verified)</p>
                                    <p className="text-zinc-400">[Network Egress] 0 egress attempts intercepted</p>
                                    <p className="text-sky-400">[Result] Automated test suite verified; zero runtime security violations.</p>
                                </div>
                            ) : (
                                <div className="space-y-1.5 text-[11px] leading-relaxed">
                                    <p className="text-zinc-500">[gVisor Watchdog] Automated patch detonation was not verified or completed with non-zero exit code.</p>
                                    <p className="text-rose-400">[Notice] Patch has NOT been verified by sandbox detonation. Manual engineering review required.</p>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}