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

    const demoScans: ScanRun[] = scans.length > 0 ? scans : [
        { id: "1", repository_id: 1, repo_name: "lucid-engine", pr_number: 442, commit_sha: "a1b2c3d4e5f6", status: "COMPLETED", findings_count: 3, scan_duration_ms: 23000, started_at: new Date(Date.now() - 3600000 * 2).toISOString(), completed_at: new Date(Date.now() - 3500000).toISOString() },
        { id: "2", repository_id: 2, repo_name: "lucid-platform", pr_number: 489, commit_sha: "f6g7h8i9j0k1", status: "COMPLETED", findings_count: 0, scan_duration_ms: 32000, started_at: new Date(Date.now() - 3600000 * 10).toISOString(), completed_at: new Date(Date.now() - 3500000 * 10).toISOString() },
        { id: "3", repository_id: 3, repo_name: "lucid-infra", pr_number: 901, commit_sha: "l2m3n4o5p6q7", status: "SCANNING", findings_count: 0, scan_duration_ms: null, started_at: new Date().toISOString() },
        { id: "4", repository_id: 1, repo_name: "lucid-engine", pr_number: 440, commit_sha: "r8s9t0u1v2w3", status: "COMPLETED", findings_count: 1, scan_duration_ms: 49000, started_at: new Date(Date.now() - 3600000 * 4).toISOString(), completed_at: new Date(Date.now() - 3500000 * 4).toISOString() },
        { id: "5", repository_id: 2, repo_name: "lucid-platform", pr_number: 487, commit_sha: "x4y5z6a7b8c9", status: "FAILED", findings_count: 0, scan_duration_ms: 5000, started_at: new Date(Date.now() - 3600000 * 24).toISOString(), completed_at: new Date(Date.now() - 3500000 * 24).toISOString() },
    ];

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
                            <th className="px-5 py-3 font-medium">Repository & PR</th>
                            <th className="px-5 py-3 font-medium">Commit SHA</th>
                            <th className="px-5 py-3 font-medium">Status</th>
                            <th className="px-5 py-3 font-medium">Findings</th>
                            <th className="px-5 py-3 font-medium">Duration</th>
                            <th className="px-5 py-3 font-medium">Timestamp</th>
                            <th className="px-5 py-3 font-medium text-right">Action</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-outline-variant/10 text-body-sm">
                        {demoScans.map((scan) => (
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
                                        <span className="text-error font-medium">{scan.findings_count} Critical</span>
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
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
