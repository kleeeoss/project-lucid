// src/app/repositories/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import {
    GitFork,
    Shield,
    Clock,
    ArrowRight,
    Search,
    Filter,
    Plus,
} from "lucide-react";

export const revalidate = 5;

async function getRepositories() {
    try {
        const res = await query(`
            SELECT r.*,
                   COUNT(DISTINCT s.id)::int as scan_count,
                   COALESCE(SUM(s.findings_count), 0)::int as total_findings,
                   MAX(s.completed_at) as last_scan_at
            FROM repositories r
            LEFT JOIN scan_runs s ON s.repository_id = r.id
            GROUP BY r.id
            ORDER BY r.created_at DESC;
        `);
        return res.rows;
    } catch {
        return [];
    }
}

function timeAgo(date: string | null) {
    if (!date) return "Never";
    const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
}

export default async function RepositoriesPage() {
    const repos = await getRepositories();

    const demoRepos = repos.length > 0 ? repos : [
        { id: 1, full_name: "lucid-engine", default_branch: "main", scan_count: 48, total_findings: 3, last_scan_at: new Date(Date.now() - 3600000).toISOString(), installation_id: 1 },
        { id: 2, full_name: "lucid-platform", default_branch: "main", scan_count: 32, total_findings: 7, last_scan_at: new Date(Date.now() - 7200000).toISOString(), installation_id: 1 },
        { id: 3, full_name: "lucid-infra", default_branch: "main", scan_count: 15, total_findings: 0, last_scan_at: new Date(Date.now() - 86400000).toISOString(), installation_id: 1 },
        { id: 4, full_name: "lucid-frontend", default_branch: "main", scan_count: 22, total_findings: 0, last_scan_at: new Date(Date.now() - 172800000).toISOString(), installation_id: 1 },
        { id: 5, full_name: "lucid-docs", default_branch: "main", scan_count: 5, total_findings: 0, last_scan_at: new Date(Date.now() - 259200000).toISOString(), installation_id: 1 },
    ];

    return (
        <div className="space-y-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-headline-xl font-bold text-on-surface">Repositories</h1>
                    <p className="mt-1 text-body-md text-on-surface-variant">
                        Manage tracked repositories and their security configurations.
                    </p>
                </div>
                <button className="flex items-center gap-2 rounded-md bg-primary-container px-3 py-1.5 text-body-sm font-semibold text-on-primary hover:bg-primary-container/90 transition shadow-lg shadow-primary-container/20">
                    <Plus className="h-4 w-4" />
                    Add Repository
                </button>
            </div>

            {/* Search and Filter */}
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant flex-1 max-w-md">
                    <Search className="h-4 w-4" />
                    <input type="text" placeholder="Search repositories..." className="bg-transparent outline-none w-full text-on-surface placeholder:text-on-surface-variant/50" />
                </div>
                <button className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant hover:bg-surface-container-high transition">
                    <Filter className="h-4 w-4" />
                    Filter
                </button>
            </div>

            {/* Repository list */}
            <div className="space-y-3">
                {demoRepos.map((repo: any) => {
                    const healthPct = Math.max(0, 100 - repo.total_findings * 10);
                    const healthColor = healthPct >= 90 ? "text-secondary" : healthPct >= 50 ? "text-warning" : "text-error";
                    const barColor = healthPct >= 90 ? "bg-secondary" : healthPct >= 50 ? "bg-warning" : "bg-error";

                    return (
                        <div key={repo.id} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-4 hover:bg-surface-container transition">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-lg bg-surface-container flex items-center justify-center">
                                        <GitFork className="h-5 w-5 text-primary" />
                                    </div>
                                    <div>
                                        <h3 className="text-body-lg font-semibold text-on-surface">{repo.full_name}</h3>
                                        <div className="flex items-center gap-3 mt-0.5 text-label-sm text-on-surface-variant">
                                            <span className="font-mono">{repo.default_branch}</span>
                                            <span>•</span>
                                            <span>{repo.scan_count} scans</span>
                                            <span>•</span>
                                            <span className="flex items-center gap-1">
                                                <Clock className="h-3 w-3" />
                                                {timeAgo(repo.last_scan_at)}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-4">
                                    <div className="text-right">
                                        <div className="flex items-center gap-2">
                                            <div className="w-20 h-1.5 rounded-full bg-surface-container overflow-hidden">
                                                <div className={`h-full rounded-full ${barColor}`} style={{ width: `${healthPct}%` }} />
                                            </div>
                                            <span className={`text-label-sm font-medium ${healthColor}`}>{healthPct}%</span>
                                        </div>
                                        <p className="text-label-sm text-on-surface-variant/60 mt-0.5">
                                            {repo.total_findings > 0 ? (
                                                <span className="text-error">{repo.total_findings} findings</span>
                                            ) : (
                                                <span className="text-secondary">Clean</span>
                                            )}
                                        </p>
                                    </div>
                                    <Link href={`/scans`} className="rounded-md border border-outline-variant/30 bg-surface-container p-2 hover:bg-surface-container-high transition">
                                        <ArrowRight className="h-4 w-4 text-on-surface-variant" />
                                    </Link>
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
