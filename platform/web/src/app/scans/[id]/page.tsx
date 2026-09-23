import Link from "next/link";
import dynamic from "next/dynamic";
import { notFound } from "next/navigation";
import { query } from "@/lib/db";
import { ScanRun, Vulnerability } from "@/lib/types";
import { StatusBadge, SeverityBadge } from "@/components/StatusBadge";
import { ArrowLeft, ShieldAlert, FileCode2, Sparkles, Check, GitCommit } from "lucide-react";

// Dynamically import React Flow with SSR disabled to prevent blank canvas hydration issues
const ASTFlowGraph = dynamic(
    () => import("@/components/ASTFlowGraph").then((mod) => mod.ASTFlowGraph),
    {
        ssr: false,
        loading: () => (
            <div className="flex h-[420px] w-full items-center justify-center rounded-xl border border-slate-800 bg-[#0a0f1d] text-xs text-slate-500">
                Loading AST Taint Flow Graph...
            </div>
        ),
    }
);

async function getScanDetails(id: string) {
    try {
        const scanRes = await query(
            `SELECT s.*, r.full_name as repo_name 
       FROM scan_runs s
       LEFT JOIN repositories r ON r.id = s.repository_id
       WHERE s.id = $1::uuid;`,
            [id]
        );

        if (scanRes.rows.length === 0) {
            console.warn("Scan run not found for ID:", id);
            return null;
        }

        const vulnsRes = await query(
            `SELECT * FROM vulnerabilities WHERE scan_run_id = $1::uuid ORDER BY line_start ASC;`,
            [id]
        );

        return {
            scan: scanRes.rows[0] as ScanRun,
            vulns: vulnsRes.rows as Vulnerability[],
        };
    } catch (error) {
        console.error("Error fetching scan detail for ID:", id, error);
        return null;
    }
}

export default async function ScanDetailPage({ params }: { params: { id: string } }) {
    const data = await getScanDetails(params.id);

    if (!data) {
        notFound();
    }

    const { scan, vulns } = data;

    return (
        <div className="space-y-8">
            {/* Back Link & Header */}
            <div>
                <Link
                    href="/"
                    className="inline-flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition"
                >
                    <ArrowLeft className="h-3.5 w-3.5" />
                    Back to Scans Feed
                </Link>
                <div className="mt-3 flex flex-wrap items-center justify-between gap-4">
                    <div>
                        <div className="flex items-center gap-3">
                            <h1 className="text-2xl font-bold tracking-tight text-white">
                                Scan Run #{scan.pr_number}
                            </h1>
                            <StatusBadge status={scan.status} />
                        </div>
                        <p className="mt-1 flex items-center gap-2 text-xs text-slate-400">
                            <span className="text-slate-300 font-medium">{scan.repo_name || "acme/lucid-ci"}</span>
                            <span>•</span>
                            <span className="flex items-center gap-1 font-mono">
                                <GitCommit className="h-3 w-3" />
                                {scan.commit_sha}
                            </span>
                        </p>
                    </div>

                    <div className="flex items-center gap-4 text-xs font-mono text-slate-400 bg-slate-900 border border-slate-800 rounded-lg px-4 py-2">
                        <div>
                            <span className="text-slate-500 block">Duration</span>
                            <span className="text-white font-semibold">
                                {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(2)}s` : "-"}
                            </span>
                        </div>
                        <div className="h-6 w-px bg-slate-800" />
                        <div>
                            <span className="text-slate-500 block">Findings</span>
                            <span className="text-rose-400 font-semibold">{vulns.length} Critical</span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Vulnerabilities Section */}
            <div className="space-y-6">
                <h2 className="text-base font-semibold text-white flex items-center gap-2">
                    <ShieldAlert className="h-4 w-4 text-rose-400" />
                    Detected Vulnerability Findings ({vulns.length})
                </h2>

                {vulns.length === 0 ? (
                    <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-12 text-center text-slate-400">
                        No vulnerabilities detected for this Pull Request.
                    </div>
                ) : (
                    vulns.map((v, index) => (
                        <div
                            key={v.id}
                            className="rounded-xl border border-slate-800 bg-slate-900/60 p-6 space-y-6 backdrop-blur-sm"
                        >
                            {/* Finding Title & Badges */}
                            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-800 pb-4">
                                <div className="flex items-center gap-3">
                                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-rose-500/20 text-xs font-bold text-rose-400">
                                        {index + 1}
                                    </span>
                                    <div>
                                        <h3 className="text-base font-semibold text-white">
                                            {v.cwe === "CWE-89" ? "SQL Injection" : v.rule_id} ({v.cwe})
                                        </h3>
                                        <p className="flex items-center gap-1.5 text-xs text-slate-400 font-mono mt-0.5">
                                            <FileCode2 className="h-3.5 w-3.5 text-slate-500" />
                                            {v.file_path}:{v.line_start}
                                        </p>
                                    </div>
                                </div>

                                <div className="flex items-center gap-2">
                                    <SeverityBadge severity={v.severity} />
                                    <span className="rounded border border-slate-800 bg-slate-800/60 px-2 py-0.5 text-xs text-slate-300">
                                        Confidence: {(Number(v.confidence_score) * 100).toFixed(0)}%
                                    </span>
                                </div>
                            </div>

                            {/* Vulnerable Code Snippet */}
                            <div>
                                <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                                    Vulnerable Code Slice
                                </span>
                                <pre className="mt-2 rounded-lg border border-red-500/30 bg-red-950/20 p-4 font-mono text-xs text-red-200 overflow-x-auto">
                                    <code>{v.vulnerable_code}</code>
                                </pre>
                            </div>

                            {/* React Flow AST Taint Graph Component (TASK-PLT-503) */}
                            <div>
                                <div className="flex items-center justify-between mb-2">
                                    <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                                        Interactive AST Taint Graph (Tree-sitter Traversal)
                                    </span>
                                    <span className="text-[11px] text-slate-500">React Flow v12 • Drag & Zoom</span>
                                </div>
                                <ASTFlowGraph vuln={v} />
                            </div>

                            {/* AI Remediation Patch */}
                            <div className="rounded-lg border border-sky-500/20 bg-sky-950/20 p-5 space-y-3">
                                <div className="flex items-center gap-2 text-sky-400">
                                    <Sparkles className="h-4 w-4" />
                                    <span className="text-xs font-bold uppercase tracking-wider">
                                        AI Remediation Patch (Small Language Model)
                                    </span>
                                </div>

                                <p className="text-xs text-slate-300 leading-relaxed">
                                    {v.ai_explanation ||
                                        "User input flows directly into query execution without parameterization. Suggested replacement uses parameterized query placeholders."}
                                </p>

                                {v.ai_remediation_patch && (
                                    <div>
                                        <div className="flex items-center justify-between py-1">
                                            <span className="text-[11px] text-slate-400 font-mono">Suggested Code Replacement:</span>
                                            <span className="text-[11px] text-emerald-400 flex items-center gap-1 font-sans">
                                                <Check className="h-3 w-3" /> Ready for 1-Click PR Merge
                                            </span>
                                        </div>
                                        <pre className="rounded border border-emerald-500/30 bg-emerald-950/20 p-3 font-mono text-xs text-emerald-300 overflow-x-auto">
                                            <code>{v.ai_remediation_patch}</code>
                                        </pre>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}