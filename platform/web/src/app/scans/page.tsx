// src/app/scans/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import { ScanRun } from "@/lib/types";
import {
    ScanSearch,
    ArrowRight,
    Clock,
    Search,
    Filter,
    Activity,
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
        <div className="space-y-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-headline-xl font-bold text-on-surface">Scan History</h1>
                    <p className="mt-1 text-body-md text-on-surface-variant">
                        Full log of all PR scan ingestion events and their results.
                    </p>
                </div>
                <div className="flex items-center gap-2 rounded-full border border-secondary/30 bg-secondary/10 px-3 py-1 text-label-sm text-secondary">
                    <span className="h-1.5 w-1.5 rounded-full bg-secondary animate-pulse" />
                    Live Auto-Refresh (5s)
                </div>
            </div>

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant flex-1 max-w-md">
                    <Search className="h-4 w-4" />
                    <input type="text" placeholder="Filter by repo, PR, or commit..." className="bg-transparent outline-none w-full text-on-surface placeholder:text-on-surface-variant/50" />
                </div>
                <button className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant hover:bg-surface-container-high transition">
                    <Filter className="h-4 w-4" />
                    All Statuses
                </button>
            </div>

            <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low overflow-hidden">
                <table className="w-full text-left">
                    <thead className="border-b border-outline-variant/20 bg-surface-dim">
                        <tr className="text-label-sm text-on-surface-variant uppercase tracking-wider">
                            <th className="px-5 py-3 font-medium">Repository &amp; PR</th>
                            <th className="px-5 py-3 font-medium">Commit SHA</th>
                            <th className="px-5 py-3 font-medium">Status</th>
                            <th className="px-5 py-3 font-medium">Findings</th>
                            <th className="px-5 py-3 font-medium">Duration</th>
                            <th className="px-5 py-3 font-medium">Timestamp</th>
                            <th className="px-5 py-3 font-medium text-right">Action</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-outline-variant/10 text-body-sm">
                        {scans.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-5 py-12 text-center text-on-surface-variant font-sans">
                                    <div className="flex flex-col items-center justify-center gap-2">
                                        <ScanSearch className="h-8 w-8 text-on-surface-variant/40" />
                                        <span className="font-medium text-on-surface">No scans recorded yet</span>
                                        <span className="text-body-sm text-on-surface-variant/70">
                                            Trigger a scan via GitHub Pull Request or webhook to view real-time analysis logs.
                                        </span>
                                    </div>
                                </td>
                            </tr>
                        ) : (
                            scans.map((scan) => (
                                <tr key={scan.id} className="scan-row">
                                    <td className="px-5 py-3">
                                        <div className="flex items-center gap-2">
                                            <span className="font-medium text-on-surface">{scan.repo_name || "lucid-ci"}</span>
                                            <span className="rounded bg-surface-container border border-outline-variant/30 px-1.5 py-0.5 font-mono text-code-sm text-primary">
                                                #{scan.pr_number}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-5 py-3 font-mono text-code-sm text-on-surface-variant">
                                        {scan.commit_sha.substring(0, 8)}
                                    </td>
                                    <td className="px-5 py-3">
                                        <StatusBadge status={scan.status} />
                                    </td>
                                    <td className="px-5 py-3">
                                        {scan.findings_count > 0 ? (
                                            <span className="text-error font-medium">
                                                {scan.findings_count} {scan.findings_count === 1 ? "finding" : "findings"}
                                            </span>
                                        ) : (
                                            <span className="text-secondary">Clean</span>
                                        )}
                                    </td>
                                    <td className="px-5 py-3 font-mono text-code-sm text-on-surface-variant">
                                        {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(1)}s` : "—"}
                                    </td>
                                    <td className="px-5 py-3 text-on-surface-variant/70">
                                        {timeAgo(scan.started_at)}
                                    </td>
                                    <td className="px-5 py-3 text-right">
                                        <Link
                                            href={`/scans/${scan.id}`}
                                            className="inline-flex items-center gap-1 rounded-md border border-outline-variant/30 bg-surface-container px-2.5 py-1 text-label-sm text-on-surface-variant hover:bg-surface-container-high transition"
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
