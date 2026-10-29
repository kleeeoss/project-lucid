// src/app/scans/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import { ScanRun } from "@/lib/types";
import {
    ScanSearch,
    ArrowRight,
    Search,
    Filter,
    Terminal,
} from "lucide-react";
import { StatusBadge } from "@/components/StatusBadge";

export const revalidate = 5;

async function getScans() {
    try {
        const res = await query(`
            SELECT s.id, s.repository_id, s.pr_number, s.commit_sha, s.status,
                   s.findings_count, s.scan_duration_ms, s.started_at, s.completed_at,
                   r.full_name as repo_name
            FROM scan_runs s
            LEFT JOIN repositories r ON r.id = s.repository_id
            ORDER BY s.started_at DESC
            LIMIT 50;
        `);
        return res.rows as ScanRun[];
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

export default async function ScansPage() {
    const scans = await getScans();

    return (
        <div className="space-y-5 px-6 py-5 max-w-7xl mx-auto w-full">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800/80 pb-5">
                <div>
                    <h1 className="text-xl font-semibold text-zinc-100 tracking-tight">Scan History</h1>
                    <p className="mt-1 text-xs text-zinc-400">
                        Full operational log of pull request scan ingestion events and AST taint analysis results.
                    </p>
                </div>
                <div className="flex items-center gap-2 rounded-md border border-emerald-500/20 bg-emerald-500/10 px-2.5 py-1 text-[11px] font-mono text-emerald-400">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    Live Polling (5s)
                </div>
            </div>

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 flex-1 max-w-md font-mono">
                    <Search className="h-3.5 w-3.5 text-zinc-500" />
                    <input
                        type="text"
                        placeholder="Filter by repo, PR, or SHA..."
                        className="bg-transparent outline-none w-full text-zinc-200 placeholder:text-zinc-600 text-xs font-mono"
                    />
                </div>
                <button className="flex items-center gap-1.5 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-mono text-zinc-300 hover:bg-zinc-800 hover:text-white transition">
                    <Filter className="h-3.5 w-3.5 text-zinc-500" />
                    All Statuses
                </button>
            </div>

            <div className="rounded-lg border border-zinc-800 bg-zinc-950/80 overflow-hidden shadow-sm">
                <table className="w-full text-left font-mono text-xs">
                    <thead className="border-b border-zinc-800/80 bg-zinc-900/60">
                        <tr className="text-[10px] text-zinc-500 uppercase tracking-wider font-semibold">
                            <th className="px-4 py-2.5">Repository &amp; PR</th>
                            <th className="px-4 py-2.5">Commit SHA</th>
                            <th className="px-4 py-2.5">Status</th>
                            <th className="px-4 py-2.5">Findings</th>
                            <th className="px-4 py-2.5">Duration</th>
                            <th className="px-4 py-2.5">Timestamp</th>
                            <th className="px-4 py-2.5 text-right font-sans">Action</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-900">
                        {scans.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-4 py-12 text-center text-zinc-500 font-sans">
                                    <div className="flex flex-col items-center justify-center gap-2">
                                        <ScanSearch className="h-8 w-8 text-zinc-600" />
                                        <span className="font-medium text-zinc-300 text-sm">No scans recorded yet</span>
                                        <span className="text-xs text-zinc-500">
                                            Trigger a scan via GitHub Pull Request or webhook to view real-time analysis logs.
                                        </span>
                                    </div>
                                </td>
                            </tr>
                        ) : (
                            scans.map((scan) => (
                                <tr key={scan.id} className="hover:bg-zinc-900/60 transition-colors">
                                    <td className="px-4 py-2.5">
                                        <div className="flex items-center gap-2">
                                            <span className="font-medium text-zinc-200">{scan.repo_name || "lucid-ci"}</span>
                                            <span className="rounded bg-zinc-900 border border-zinc-800 px-1.5 py-0.5 text-[10px] text-sky-400">
                                                #{scan.pr_number}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-4 py-2.5 text-zinc-400">
                                        {scan.commit_sha ? scan.commit_sha.substring(0, 8) : "—"}
                                    </td>
                                    <td className="px-4 py-2.5">
                                        <StatusBadge status={scan.status} />
                                    </td>
                                    <td className="px-4 py-2.5">
                                        {scan.findings_count > 0 ? (
                                            <span className="text-rose-400 font-semibold">
                                                {scan.findings_count} {scan.findings_count === 1 ? "finding" : "findings"}
                                            </span>
                                        ) : (
                                            <span className="text-emerald-400">0 clean</span>
                                        )}
                                    </td>
                                    <td className="px-4 py-2.5 text-zinc-500">
                                        {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(1)}s` : "—"}
                                    </td>
                                    <td className="px-4 py-2.5 text-zinc-500">
                                        {timeAgo(scan.started_at)}
                                    </td>
                                    <td className="px-4 py-2.5 text-right font-sans">
                                        <Link
                                            href={`/scans/${scan.id}`}
                                            className="inline-flex items-center gap-1 rounded-md border border-zinc-800 bg-zinc-900 px-2.5 py-1 text-xs text-zinc-300 hover:bg-zinc-800 hover:text-white transition"
                                        >
                                            Inspect <ArrowRight className="h-3 w-3" />
                                        </Link>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
