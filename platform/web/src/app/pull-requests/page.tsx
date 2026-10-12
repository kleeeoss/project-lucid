// src/app/pull-requests/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import {
    GitPullRequest,
    CheckCircle2,
    AlertTriangle,
    XCircle,
    Clock,
    Search,
    Filter,
    ArrowRight,
} from "lucide-react";

export const revalidate = 5;

async function getPullRequests() {
    try {
        const res = await query(`
            SELECT s.id, s.pr_number, s.commit_sha, s.status, s.findings_count,
                   s.scan_duration_ms, s.started_at, s.completed_at,
                   r.full_name as repo_name
            FROM scan_runs s
            LEFT JOIN repositories r ON r.id = s.repository_id
            WHERE s.pr_number IS NOT NULL
            ORDER BY s.started_at DESC
            LIMIT 25;
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

export default async function PullRequestsPage() {
    const prs = await getPullRequests();

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-headline-xl font-bold text-on-surface">Pull Requests</h1>
                <p className="mt-1 text-body-md text-on-surface-variant">
                    Security gate results for all scanned pull requests.
                </p>
            </div>

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant flex-1 max-w-md">
                    <Search className="h-4 w-4" />
                    <input type="text" placeholder="Search PRs by number, repo, or author..." className="bg-transparent outline-none w-full text-on-surface placeholder:text-on-surface-variant/50" />
                </div>
                <button className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant hover:bg-surface-container-high transition">
                    <Filter className="h-4 w-4" />
                    All Statuses
                </button>
            </div>

            {prs.length === 0 ? (
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-12 text-center">
                    <GitPullRequest className="h-10 w-10 text-on-surface-variant/40 mx-auto mb-3" />
                    <h3 className="text-body-lg font-semibold text-on-surface">No pull requests evaluated yet</h3>
                    <p className="mt-1 text-body-sm text-on-surface-variant/70 max-w-sm mx-auto">
                        When pull requests are opened or synchronized on monitored repositories, security gate evaluations will appear here.
                    </p>
                </div>
            ) : (
                <div className="space-y-2">
                    {prs.map((pr: any) => (
                        <Link
                            key={pr.id}
                            href={`/scans/${pr.id}`}
                            className="block rounded-lg border border-outline-variant/30 bg-surface-container-low p-4 hover:bg-surface-container transition"
                        >
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    {pr.status === "COMPLETED" && pr.findings_count === 0 ? (
                                        <CheckCircle2 className="h-5 w-5 text-secondary flex-shrink-0" />
                                    ) : pr.status === "FAILED" || pr.findings_count > 0 ? (
                                        <AlertTriangle className="h-5 w-5 text-warning flex-shrink-0" />
                                    ) : pr.status === "SCANNING" ? (
                                        <Clock className="h-5 w-5 text-primary animate-pulse flex-shrink-0" />
                                    ) : (
                                        <GitPullRequest className="h-5 w-5 text-on-surface-variant flex-shrink-0" />
                                    )}
                                    <div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-body-lg font-semibold text-on-surface">
                                                PR #{pr.pr_number}
                                            </span>
                                            <span className="font-mono text-code-sm text-primary">{pr.repo_name}</span>
                                        </div>
                                        <div className="flex items-center gap-3 mt-0.5 text-label-sm text-on-surface-variant">
                                            <span className="font-mono">{pr.commit_sha?.substring(0, 8)}</span>
                                            <span>•</span>
                                            <span>{timeAgo(pr.started_at)}</span>
                                            {pr.scan_duration_ms && (
                                                <>
                                                    <span>•</span>
                                                    <span>{(pr.scan_duration_ms / 1000).toFixed(0)}s</span>
                                                </>
                                            )}
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-3">
                                    {pr.findings_count > 0 && (
                                        <span className="rounded border border-error/30 bg-error/10 px-2 py-0.5 text-label-sm text-error font-medium">
                                            {pr.findings_count} findings
                                        </span>
                                    )}
                                    <span
                                        className={`rounded border px-2 py-0.5 text-label-sm font-medium ${
                                            pr.status === "COMPLETED"
                                                ? "bg-secondary/10 text-secondary border-secondary/30"
                                                : pr.status === "SCANNING"
                                                ? "bg-primary/10 text-primary border-primary/30"
                                                : "bg-error/10 text-error border-error/30"
                                        }`}
                                    >
                                        {pr.status}
                                    </span>
                                    <ArrowRight className="h-4 w-4 text-on-surface-variant/50" />
                                </div>
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}
