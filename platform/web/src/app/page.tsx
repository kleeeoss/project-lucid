// src/app/page.tsx
import Link from "next/link";
import { query } from "@/lib/db";
import { ScanRun } from "@/lib/types";
import { MetricCard } from "@/components/MetricCard";
import { StatusBadge } from "@/components/StatusBadge";
import { ShieldAlert, CheckCircle2, GitPullRequest, Clock, ArrowRight, Activity, Search } from "lucide-react";

export const revalidate = 5; // Live auto-polling every 5 seconds

async function getDashboardData() {
  try {
    const scansRes = await query(`
      SELECT s.id, s.repository_id, s.pr_number, s.commit_sha, s.status, 
             s.findings_count, s.scan_duration_ms, s.started_at, s.completed_at,
             r.full_name as repo_name
      FROM scan_runs s
      LEFT JOIN repositories r ON r.id = s.repository_id
      ORDER BY s.started_at DESC
      LIMIT 25;
    `);

    const statsRes = await query(`
      SELECT 
        COUNT(*)::int as total_scans,
        COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END)::int as completed_scans,
        COALESCE(SUM(findings_count), 0)::int as total_findings,
        COALESCE(AVG(scan_duration_ms), 0)::int as avg_duration
      FROM scan_runs;
    `);

    const stats = statsRes.rows[0] || {
      total_scans: 0,
      completed_scans: 0,
      total_findings: 0,
      avg_duration: 0,
    };

    return {
      scans: scansRes.rows as ScanRun[],
      stats: {
        totalScans: stats.total_scans,
        completedScans: stats.completed_scans,
        criticalVulns: stats.total_findings,
        avgDurationMs: stats.avg_duration,
      },
    };
  } catch (error) {
    console.error("Database query failed:", error);
    return {
      scans: [],
      stats: { totalScans: 0, completedScans: 0, criticalVulns: 0, avgDurationMs: 0 },
    };
  }
}

export default async function DashboardPage() {
  const { scans, stats } = await getDashboardData();

  return (
    <div className="space-y-8">
      {/* Top Header with Live Ingestion Indicator */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-800 pb-5">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">DevSecOps Control Plane</h1>
          <p className="mt-1 text-xs text-zinc-400">
            Real-time PR interception, Tree-sitter taint analysis, and gVisor detonation verification.
          </p>
        </div>

        <div className="flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-950/30 px-3 py-1 text-xs text-emerald-400 font-mono">
          <span className="h-2 w-2 rounded-full bg-emerald-400 animate-ping" />
          <span>Pipeline Nominal • SQS Polling Active</span>
        </div>
      </div>

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Scans Processed"
          value={stats.totalScans}
          subtitle="Triggered via GitHub Webhooks"
          icon={<GitPullRequest className="h-5 w-5 text-sky-400" />}
        />
        <MetricCard
          title="Vulnerabilities Intercepted"
          value={stats.criticalVulns}
          subtitle="SQLi, Cmd Injection & Secrets"
          icon={<ShieldAlert className="h-5 w-5 text-rose-400" />}
        />
        <MetricCard
          title="PR Check Runs Published"
          value={stats.completedScans}
          subtitle="Annotations & 1-Click Suggestions"
          icon={<CheckCircle2 className="h-5 w-5 text-emerald-400" />}
        />
        <MetricCard
          title="Mean Analysis Latency"
          value={`${(stats.avgDurationMs / 1000).toFixed(1)}s`}
          subtitle="Sub-minute CI SLA Budget"
          icon={<Clock className="h-5 w-5 text-amber-400" />}
        />
      </div>

      {/* Scans Feed Table */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/60 overflow-hidden backdrop-blur-md shadow-xl">
        <div className="border-b border-zinc-800 px-6 py-4 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-sky-400" />
            <h2 className="text-sm font-semibold text-white">Pull Request Scan Ingestion Stream</h2>
          </div>
          <span className="text-[11px] font-mono text-zinc-500">Live Auto-Refresh Active (5s)</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-zinc-800 bg-zinc-950/80 text-[11px] font-bold text-zinc-400 uppercase tracking-wider">
              <tr>
                <th className="px-6 py-3.5">Repository & PR</th>
                <th className="px-6 py-3.5">Commit SHA</th>
                <th className="px-6 py-3.5">Status</th>
                <th className="px-6 py-3.5">Vulnerability Count</th>
                <th className="px-6 py-3.5">Pipeline Latency</th>
                <th className="px-6 py-3.5">Timestamp</th>
                <th className="px-6 py-3.5 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/60 font-mono text-xs">
              {scans.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-zinc-500 font-sans">
                    No scans recorded in PostgreSQL store yet.
                  </td>
                </tr>
              ) : (
                scans.map((scan) => (
                  <tr key={scan.id} className="hover:bg-zinc-800/40 transition">
                    <td className="px-6 py-4 font-sans font-medium text-white">
                      <div className="flex items-center gap-2">
                        <span className="text-zinc-200">{scan.repo_name || "acme/lucid-ci"}</span>
                        <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[11px] font-mono text-sky-400 border border-zinc-700">
                          #{scan.pr_number}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-zinc-400">
                      {scan.commit_sha.substring(0, 8)}
                    </td>
                    <td className="px-6 py-4 font-sans">
                      <StatusBadge status={scan.status} />
                    </td>
                    <td className="px-6 py-4 font-sans">
                      {scan.findings_count > 0 ? (
                        <span className="inline-flex items-center gap-1.5 font-bold text-rose-400">
                          <span className="h-1.5 w-1.5 rounded-full bg-rose-400" />
                          {scan.findings_count} Critical
                        </span>
                      ) : (
                        <span className="text-emerald-400 font-medium">Clean</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-zinc-400">
                      {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(2)}s` : "-"}
                    </td>
                    <td className="px-6 py-4 text-zinc-500 font-sans">
                      {new Date(scan.started_at).toLocaleTimeString()}
                    </td>
                    <td className="px-6 py-4 text-right font-sans">
                      <Link
                        href={`/scans/${scan.id}`}
                        className="inline-flex items-center gap-1 rounded-md border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs font-medium text-zinc-200 hover:bg-zinc-700 hover:text-white transition"
                      >
                        Inspect Workbench
                        <ArrowRight className="h-3 w-3" />
                      </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}