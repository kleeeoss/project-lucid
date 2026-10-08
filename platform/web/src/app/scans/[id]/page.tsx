// src/app/scans/[id]/page.tsx
import Link from "next/link";
import { notFound } from "next/navigation";
import { query } from "@/lib/db";
import { ScanRun, Vulnerability } from "@/lib/types";
import { ScanWorkbench } from "@/components/scan/ScanWorkbench";
import { ArrowLeft, GitCommit, GitPullRequest, Clock, ShieldAlert } from "lucide-react";
import { StatusBadge } from "@/components/StatusBadge";

async function getScanDetails(id: string) {
    try {
        const scanRes = await query(
            `SELECT s.*, r.full_name as repo_name 
       FROM scan_runs s
       LEFT JOIN repositories r ON r.id = s.repository_id
       WHERE s.id = $1::uuid;`,
            [id]
        );

        if (scanRes.rows.length === 0) return null;

        const vulnsRes = await query(
            `SELECT * FROM vulnerabilities WHERE scan_run_id = $1::uuid ORDER BY 
       CASE severity 
         WHEN 'CRITICAL' THEN 1 
         WHEN 'HIGH' THEN 2 
         WHEN 'MEDIUM' THEN 3 
         ELSE 4 
       END ASC, line_start ASC;`,
            [id]
        );

        return {
            scan: scanRes.rows[0] as ScanRun,
            vulns: vulnsRes.rows as Vulnerability[],
        };
    } catch (error) {
        console.error("Error fetching scan detail:", error);
        return null;
    }
}

export default async function ScanDetailPage({ params }: { params: { id: string } }) {
    const data = await getScanDetails(params.id);
    if (!data) notFound();

    const { scan, vulns } = data;

    return (
        <div className="space-y-6">
            {/* Top Breadcrumb & Metadata Header */}
            <div className="rounded-xl border border-zinc-800/80 bg-zinc-900/60 p-5 backdrop-blur-md">
                <div className="flex flex-wrap items-center justify-between gap-4">
                    <div>
                        <Link
                            href="/"
                            className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-400 hover:text-white transition"
                        >
                            <ArrowLeft className="h-3.5 w-3.5" />
                            Back to Scans Feed
                        </Link>

                        <div className="mt-3 flex items-center gap-3">
                            <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
                                <GitPullRequest className="h-5 w-5 text-sky-400" />
                                {scan.repo_name || "acme/lucid-ci"}
                                <span className="text-zinc-500 font-normal">/</span>
                                <span className="text-sky-400">PR #{scan.pr_number}</span>
                            </h1>
                            <StatusBadge status={scan.status} />
                        </div>

                        <div className="mt-2 flex flex-wrap items-center gap-4 text-xs font-mono text-zinc-400">
                            <span className="flex items-center gap-1.5 bg-zinc-800/60 px-2 py-0.5 rounded border border-zinc-700/50">
                                <GitCommit className="h-3.5 w-3.5 text-zinc-500" />
                                {scan.commit_sha.substring(0, 10)}
                            </span>
                            <span>•</span>
                            <span className="flex items-center gap-1">
                                <Clock className="h-3.5 w-3.5 text-zinc-500" />
                                Duration: {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(2)}s` : "-"}
                            </span>
                            <span>•</span>
                            <span className="text-zinc-500">
                                Detonated in Docker-on-EC2 with gVisor (<code className="text-emerald-400">runsc</code>)
                            </span>
                        </div>
                    </div>

                    <div className="flex items-center gap-3">
                        <div className="text-right">
                            <span className="text-[10px] uppercase font-bold tracking-wider text-zinc-500 block">
                                Security Posture
                            </span>
                            <span className="text-sm font-semibold text-rose-400 flex items-center gap-1.5 justify-end">
                                <ShieldAlert className="h-4 w-4" />
                                {vulns.length} {vulns.length === 1 ? "Vulnerability" : "Vulnerabilities"} Flagged
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Interactive Master-Detail Client Component */}
            <ScanWorkbench vulns={vulns} scan={scan} />
        </div>
    );
}