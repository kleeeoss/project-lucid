// src/app/findings/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import {
    ShieldAlert,
    Search,
    Filter,
    ArrowRight,
    CheckCircle2,
} from "lucide-react";
import { SeverityBadge } from "@/components/StatusBadge";

export const revalidate = 5;

async function getFindings() {
    try {
        const res = await query(`
            SELECT v.*, r.full_name as repo_name, s.pr_number
            FROM vulnerabilities v
            LEFT JOIN scan_runs s ON s.id = v.scan_run_id
            LEFT JOIN repositories r ON r.id = s.repository_id
            ORDER BY
                CASE v.severity
                    WHEN 'CRITICAL' THEN 1
                    WHEN 'HIGH' THEN 2
                    WHEN 'MEDIUM' THEN 3
                    WHEN 'LOW' THEN 4
                    ELSE 5
                END ASC,
                v.created_at DESC
            LIMIT 50;
        `);
        return res.rows;
    } catch {
        return [];
    }
}

function timeAgo(date: string | null) {
    if (!date) return "—";
    const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
}

export default async function FindingsPage() {
    const findings = await getFindings();

    return (
        <div className="space-y-5 px-6 py-5 max-w-7xl mx-auto w-full">
            <div className="border-b border-zinc-800/80 pb-5">
                <h1 className="text-xl font-semibold text-zinc-100 tracking-tight">Vulnerability Findings</h1>
                <p className="mt-1 text-xs text-zinc-400">
                    All detected AST taint vulnerabilities across repositories, prioritized by CWE exploitability.
                </p>
            </div>

            {/* Stats row */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 font-mono">
                {[
                    { label: "Critical", count: findings.filter((f: any) => f.severity === "CRITICAL").length, color: "text-rose-400 border-rose-500/20 bg-rose-500/10" },
                    { label: "High", count: findings.filter((f: any) => f.severity === "HIGH").length, color: "text-amber-400 border-amber-500/20 bg-amber-500/10" },
                    { label: "Medium", count: findings.filter((f: any) => f.severity === "MEDIUM").length, color: "text-indigo-400 border-indigo-500/20 bg-indigo-500/10" },
                    { label: "Low", count: findings.filter((f: any) => f.severity === "LOW").length, color: "text-zinc-400 border-zinc-800 bg-zinc-900/60" },
                ].map((s) => (
                    <div key={s.label} className={`rounded-lg border p-3 text-center ${s.color}`}>
                        <p className="text-2xl font-bold">{s.count}</p>
                        <p className="text-[11px] uppercase tracking-wider">{s.label}</p>
                    </div>
                ))}
            </div>

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 flex-1 max-w-md font-mono">
                    <Search className="h-3.5 w-3.5 text-zinc-500" />
                    <input
                        type="text"
                        placeholder="Search by CWE, file, or rule..."
                        className="bg-transparent outline-none w-full text-zinc-200 placeholder:text-zinc-600 text-xs font-mono"
                    />
                </div>
                <button className="flex items-center gap-1.5 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-mono text-zinc-300 hover:bg-zinc-800 hover:text-white transition">
                    <Filter className="h-3.5 w-3.5 text-zinc-500" />
                    All Severities
                </button>
            </div>

            {findings.length === 0 ? (
                <div className="rounded-lg border border-zinc-800 bg-zinc-950/80 p-12 text-center">
                    <ShieldAlert className="h-10 w-10 text-zinc-600 mx-auto mb-3" />
                    <h3 className="text-sm font-semibold text-zinc-200">No vulnerabilities detected</h3>
                    <p className="mt-1 text-xs text-zinc-500 max-w-sm mx-auto">
                        All scanned repositories and pull requests are verified clean. Detected issues will appear here with AST taint paths.
                    </p>
                </div>
            ) : (
                <div className="space-y-2">
                    {findings.map((finding: any) => (
                        <div
                            key={finding.id}
                            className="rounded-lg border border-zinc-800 bg-zinc-950/80 p-4 hover:bg-zinc-900/60 transition"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex items-start gap-3 flex-1 min-w-0">
                                    <SeverityBadge severity={finding.severity} />
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 flex-wrap font-mono">
                                            <span className="text-xs font-semibold text-zinc-100">
                                                {finding.rule_id}
                                            </span>
                                            <code className="text-[10px] text-zinc-400 bg-zinc-900 border border-zinc-800 px-1.5 py-0.5 rounded">
                                                {finding.cwe}
                                            </code>
                                            {finding.sandbox_verified && (
                                                <span className="rounded bg-emerald-500/10 border border-emerald-500/20 px-1.5 py-0.5 text-[10px] text-emerald-400 font-medium">
                                                    gVisor Verified
                                                </span>
                                            )}
                                        </div>
                                        <p className="mt-1 text-xs text-zinc-300 font-sans">{finding.ai_explanation}</p>
                                        <div className="flex items-center gap-3 mt-2 text-[11px] font-mono text-zinc-500">
                                            <code className="text-rose-400">{finding.file_path}:{finding.line_start}</code>
                                            <span>•</span>
                                            <span>{finding.repo_name}</span>
                                            <span>•</span>
                                            <span>PR #{finding.pr_number}</span>
                                            <span>•</span>
                                            <span>{timeAgo(finding.created_at)}</span>
                                            <span>•</span>
                                            <span>Confidence: {Math.round((Number(finding.confidence_score) || 0) * 100)}%</span>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 flex-shrink-0">
                                    <Link
                                        href={`/scans/${finding.scan_run_id}`}
                                        className="inline-flex items-center gap-1 rounded border border-zinc-800 bg-zinc-900 px-2.5 py-1 text-xs text-zinc-300 hover:bg-zinc-800 hover:text-white transition font-sans"
                                    >
                                        Inspect <ArrowRight className="h-3 w-3" />
                                    </Link>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
