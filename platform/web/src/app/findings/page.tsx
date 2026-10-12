// src/app/findings/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import { Vulnerability } from "@/lib/types";
import {
    ShieldAlert,
    Search,
    Filter,
    Eye,
    Copy,
    ArrowRight,
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
        <div className="space-y-6">
            <div>
                <h1 className="text-headline-xl font-bold text-on-surface">Findings</h1>
                <p className="mt-1 text-body-md text-on-surface-variant">
                    All detected vulnerabilities across repositories, ordered by exploitability.
                </p>
            </div>

            {/* Stats row */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {[
                    { label: "Critical", count: findings.filter((f: any) => f.severity === "CRITICAL").length, color: "text-error border-error/30 bg-error/10" },
                    { label: "High", count: findings.filter((f: any) => f.severity === "HIGH").length, color: "text-orange-400 border-orange-500/30 bg-orange-500/10" },
                    { label: "Medium", count: findings.filter((f: any) => f.severity === "MEDIUM").length, color: "text-primary border-primary/30 bg-primary/10" },
                    { label: "Low", count: findings.filter((f: any) => f.severity === "LOW").length, color: "text-on-surface-variant border-outline-variant/30 bg-surface-container" },
                ].map((s) => (
                    <div key={s.label} className={`rounded-lg border p-3 text-center ${s.color}`}>
                        <p className="text-2xl font-bold">{s.count}</p>
                        <p className="text-label-sm">{s.label}</p>
                    </div>
                ))}
            </div>

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant flex-1 max-w-md">
                    <Search className="h-4 w-4" />
                    <input type="text" placeholder="Search by CWE, file, or rule..." className="bg-transparent outline-none w-full text-on-surface placeholder:text-on-surface-variant/50" />
                </div>
                <button className="flex items-center gap-2 rounded-md border border-outline-variant/30 bg-surface-container px-3 py-2 text-body-sm text-on-surface-variant hover:bg-surface-container-high transition">
                    <Filter className="h-4 w-4" />
                    All Severities
                </button>
            </div>

            {findings.length === 0 ? (
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-12 text-center">
                    <ShieldAlert className="h-10 w-10 text-on-surface-variant/40 mx-auto mb-3" />
                    <h3 className="text-body-lg font-semibold text-on-surface">No vulnerabilities detected</h3>
                    <p className="mt-1 text-body-sm text-on-surface-variant/70 max-w-sm mx-auto">
                        All scanned repositories and PRs are currently clean. Detected issues will appear here with exploitability classifications.
                    </p>
                </div>
            ) : (
                <div className="space-y-2">
                    {findings.map((finding: any) => (
                        <div
                            key={finding.id}
                            className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-4 hover:bg-surface-container transition"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex items-start gap-3 flex-1">
                                    <SeverityBadge severity={finding.severity} />
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 flex-wrap">
                                            <span className="text-body-md font-semibold text-on-surface">
                                                {finding.rule_id.replace(/-/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase())}
                                            </span>
                                            <code className="text-code-sm text-on-surface-variant font-mono bg-surface-container px-1.5 py-0.5 rounded">
                                                {finding.cwe}
                                            </code>
                                            {finding.sandbox_verified && (
                                                <span className="rounded bg-error/10 border border-error/30 px-1.5 py-0.5 text-label-sm text-error font-medium">
                                                    Sandbox Verified
                                                </span>
                                            )}
                                        </div>
                                        <p className="mt-1 text-body-sm text-on-surface-variant">{finding.ai_explanation}</p>
                                        <div className="flex items-center gap-3 mt-2 text-label-sm text-on-surface-variant/60">
                                            <code className="font-mono text-primary">{finding.file_path}:{finding.line_start}</code>
                                            <span>•</span>
                                            <span>{finding.repo_name}</span>
                                            <span>•</span>
                                            <span>PR #{finding.pr_number}</span>
                                            <span>•</span>
                                            <span>{timeAgo(finding.created_at)}</span>
                                            <span>•</span>
                                            <span>Confidence: {Math.round(finding.confidence_score * 100)}%</span>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 flex-shrink-0">
                                    <Link
                                        href={`/scans/${finding.scan_run_id}`}
                                        className="inline-flex items-center gap-1 rounded border border-outline-variant/30 bg-surface-container px-2.5 py-1 text-label-sm text-on-surface-variant hover:bg-surface-container-high transition"
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
