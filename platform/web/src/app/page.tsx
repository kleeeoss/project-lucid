import Link from "next/link";
import { query } from "@/lib/db";
import { ScanRun } from "@/lib/types";
import { MetricCard } from "@/components/MetricCard";
import { StatusBadge } from "@/components/StatusBadge";
import { ShieldAlert, CheckCircle2, Cpu, Clock, GitPullRequest, ArrowRight } from "lucide-react";

async function getDashboardData() {
  try {
    const scansRes = await query(`
      SELECT s.id, s.repository_id, s.pr_number, s.commit_sha, s.status, 
             s.findings_count, s.scan_duration_ms, s.started_at, s.completed_at,
             r.full_name as repo_name
      FROM scan_runs s
      LEFT JOIN repositories r ON r.id = s.repository_id
      ORDER BY s.started_at DESC
      LIMIT 20;
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
      {/* Top Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-white">Security Scan Operations</h1>
        <p className="mt-1 text-sm text-slate-400">
          Live monitoring of Pull Requests, AST taint traces, and Small Language Model patch generation.
        </p>
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
          title="Vulnerabilities Blocked"
          value={stats.criticalVulns}
          subtitle="SQLi, Cmd Injection & Secrets"
          icon={<ShieldAlert className="h-5 w-5 text-rose-400" />}
        />
        <MetricCard
          title="Successful Cycles"
          value={stats.completedScans}
          subtitle="Check Runs finalized"
          icon={<CheckCircle2 className="h-5 w-5 text-emerald-400" />}
        />
        <MetricCard
          title="Avg Cycle Latency"
          value={`${(stats.avgDurationMs / 1000).toFixed(1)}s`}
          subtitle="Target SLA: < 120 seconds"
          icon={<Clock className="h-5 w-5 text-amber-400" />}
        />
      </div>

      {/* Scans Feed Table */}
      <div className="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden backdrop-blur-sm">
        <div className="border-b border-slate-800 px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Cpu className="h-4 w-4 text-sky-400" />
            <h2 className="text-sm font-semibold text-white">Recent Pull Request Scans</h2>
          </div>
          <span className="text-xs text-slate-400">Auto-refreshing PostgreSQL Store</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-800 bg-slate-900/90 text-xs font-semibold text-slate-400 uppercase tracking-wider">
              <tr>
                <th className="px-6 py-3">Repository & PR</th>
                <th className="px-6 py-3">Commit SHA</th>
                <th className="px-6 py-3">Status</th>
                <th className="px-6 py-3">Findings</th>
                <th className="px-6 py-3">Duration</th>
                <th className="px-6 py-3">Timestamp</th>
                <th className="px-6 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono text-xs">
              {scans.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-slate-500 font-sans">
                    No scan runs found in PostgreSQL. Send a webhook to populate.
                  </td>
                </tr>
              ) : (
                scans.map((scan) => (
                  <tr key={scan.id} className="hover:bg-slate-800/40 transition">
                    <td className="px-6 py-4 font-sans font-medium text-white">
                      <div className="flex items-center gap-2">
                        <span className="text-slate-300">{scan.repo_name || "acme/lucid-ci"}</span>
                        <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-sky-400">
                          #{scan.pr_number}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-slate-400">
                      {scan.commit_sha.substring(0, 8)}
                    </td>
                    <td className="px-6 py-4 font-sans">
                      <StatusBadge status={scan.status} />
                    </td>
                    <td className="px-6 py-4 font-sans">
                      {scan.findings_count > 0 ? (
                        <span className="inline-flex items-center gap-1 font-semibold text-rose-400">
                          <span className="h-1.5 w-1.5 rounded-full bg-rose-400" />
                          {scan.findings_count} Finding
                        </span>
                      ) : (
                        <span className="text-slate-400">Clean</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-slate-400">
                      {scan.scan_duration_ms ? `${(scan.scan_duration_ms / 1000).toFixed(1)}s` : "-"}
                    </td>
                    <td className="px-6 py-4 text-slate-500 font-sans">
                      {new Date(scan.started_at).toLocaleTimeString()}
                    </td>
                    <td className="px-6 py-4 text-right font-sans">
                      <Link
                        href={`/scans/${scan.id}`}
                        className="inline-flex items-center gap-1 rounded-md border border-slate-700 bg-slate-800 px-2.5 py-1 text-xs font-medium text-slate-200 hover:bg-slate-700 hover:text-white transition"
                      >
                        Inspect
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
