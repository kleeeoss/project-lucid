// src/app/remediation/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import {
    Wrench,
    CheckCircle2,
    Clock,
    ArrowRight,
    Sparkles,
} from "lucide-react";

export const revalidate = 5;

async function getRemediations() {
    try {
        const res = await query(`
            SELECT v.id, v.scan_run_id, v.rule_id, v.cwe, v.severity, v.confidence_score,
                   v.file_path, v.line_start, v.line_end, v.vulnerable_code,
                   v.ai_remediation_patch, v.ai_explanation, v.sandbox_verified, v.created_at,
                   r.full_name as repo_name, s.pr_number
            FROM vulnerabilities v
            LEFT JOIN scan_runs s ON s.id = v.scan_run_id
            LEFT JOIN repositories r ON r.id = s.repository_id
            WHERE v.ai_remediation_patch IS NOT NULL AND v.ai_remediation_patch != ''
            ORDER BY v.created_at DESC
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

export default async function RemediationPage() {
    const remediations = await getRemediations();

    const severityColors: Record<string, string> = {
        CRITICAL: "bg-error text-on-error",
        HIGH: "bg-orange-500/20 text-orange-400 border border-orange-500/30",
        MEDIUM: "bg-primary/15 text-primary border border-primary/30",
        LOW: "bg-surface-container text-on-surface-variant border border-outline-variant/30",
    };

    const verifiedCount = remediations.filter((r: any) => r.sandbox_verified).length;
    const pendingCount = remediations.filter((r: any) => !r.sandbox_verified).length;

    return (
        <div className="space-y-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-headline-xl font-bold text-on-surface">Remediation</h1>
                    <p className="mt-1 text-body-md text-on-surface-variant">
                        AI-generated patches and sandbox verification results for detected vulnerabilities.
                    </p>
                </div>
                <div className="flex items-center gap-2 text-label-sm text-secondary border border-secondary/30 bg-secondary/10 px-3 py-1 rounded-full">
                    <Sparkles className="h-4 w-4" />
                    <span>AI Remediation Pipeline Active</span>
                </div>
            </div>

            {/* Summary stats */}
            <div className="grid grid-cols-3 gap-3">
                <div className="rounded-lg border border-primary/30 bg-primary/10 p-4 text-center">
                    <p className="text-2xl font-bold text-primary">{remediations.length}</p>
                    <p className="text-label-sm text-primary">Total Patches</p>
                </div>
                <div className="rounded-lg border border-secondary/30 bg-secondary/10 p-4 text-center">
                    <p className="text-2xl font-bold text-secondary">{verifiedCount}</p>
                    <p className="text-label-sm text-secondary">Sandbox Verified</p>
                </div>
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container p-4 text-center">
                    <p className="text-2xl font-bold text-on-surface-variant">{pendingCount}</p>
                    <p className="text-label-sm text-on-surface-variant">Pending Verification</p>
                </div>
            </div>

            {remediations.length === 0 ? (
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-12 text-center">
                    <Wrench className="h-10 w-10 text-on-surface-variant/40 mx-auto mb-3" />
                    <h3 className="text-body-lg font-semibold text-on-surface">No AI remediation patches generated yet</h3>
                    <p className="mt-1 text-body-sm text-on-surface-variant/70 max-w-sm mx-auto">
                        When the AST engine identifies vulnerabilities and the AI pipeline generates patches, persisted diffs and gVisor sandbox verification results will appear here.
                    </p>
                </div>
            ) : (
                <div className="space-y-3">
                    {remediations.map((rem: any) => (
                        <div key={rem.id} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-4 hover:bg-surface-container transition">
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <div className="flex items-center gap-2 flex-wrap">
                                        <span className={`rounded px-1.5 py-0.5 text-label-sm font-bold ${severityColors[rem.severity] || "bg-surface-container text-on-surface"}`}>
                                            {rem.severity}
                                        </span>
                                        <h3 className="text-body-lg font-semibold text-on-surface">
                                            {rem.rule_id ? rem.rule_id.replace(/-/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase()) : "Vulnerability"} in {rem.file_path}:{rem.line_start}
                                        </h3>
                                        <code className="text-code-sm text-on-surface-variant font-mono bg-surface-container px-1.5 py-0.5 rounded">
                                            {rem.cwe}
                                        </code>
                                    </div>

                                    <div className="mt-2 rounded-md bg-surface-dim border border-outline-variant/20 px-3 py-2">
                                        <p className="text-code-sm font-mono text-on-surface-variant overflow-x-auto whitespace-pre-wrap">
                                            <span className="text-secondary font-bold">Suggested Patch:</span> {rem.ai_remediation_patch}
                                        </p>
                                    </div>

                                    <div className="flex items-center gap-3 mt-2 text-label-sm text-on-surface-variant/60 flex-wrap">
                                        <span>{rem.repo_name || "lucid-ci"}</span>
                                        <span>•</span>
                                        <span>PR #{rem.pr_number}</span>
                                        <span>•</span>
                                        <span>Confidence: {Math.round(rem.confidence_score * 100)}%</span>
                                        <span>•</span>
                                        <span>{timeAgo(rem.created_at)}</span>
                                    </div>
                                </div>
                                <div className="flex flex-col items-end gap-2 flex-shrink-0">
                                    {rem.sandbox_verified ? (
                                        <span className="inline-flex items-center gap-1 rounded border border-secondary/30 bg-secondary/10 px-2 py-0.5 text-label-sm text-secondary font-medium">
                                            <CheckCircle2 className="h-3 w-3" />
                                            Sandbox Verified
                                        </span>
                                    ) : (
                                        <span className="inline-flex items-center gap-1 rounded border border-warning/30 bg-warning/10 px-2 py-0.5 text-label-sm text-warning font-medium">
                                            <Clock className="h-3 w-3" />
                                            Pending Sandbox Verification
                                        </span>
                                    )}
                                    <Link
                                        href={`/scans/${rem.scan_run_id}`}
                                        className="flex items-center gap-1 rounded bg-primary-container px-3 py-1.5 text-label-sm font-semibold text-on-primary hover:bg-primary-container/90 transition"
                                    >
                                        Inspect Workbench <ArrowRight className="h-3 w-3" />
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
