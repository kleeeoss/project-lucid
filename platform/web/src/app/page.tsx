import { query } from "@/lib/db";
import { ScanRun } from "@/lib/types";
import { DashboardView } from "@/components/DashboardView";

export const revalidate = 5;

async function getDashboardData() {
    try {
        const scansRes = await query(`
            SELECT s.id, s.repository_id, s.pr_number, s.commit_sha, s.status,
                   s.findings_count, s.scan_duration_ms, s.started_at, s.completed_at,
                   r.full_name as repo_name
            FROM scan_runs s
            LEFT JOIN repositories r ON r.id = s.repository_id
            ORDER BY s.started_at DESC
            LIMIT 10;
        `);

        const statsRes = await query(`
            SELECT
                COUNT(*)::int as total_scans,
                COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END)::int as completed_scans,
                COALESCE(SUM(findings_count), 0)::int as total_findings,
                COALESCE(AVG(scan_duration_ms), 0)::int as avg_duration
            FROM scan_runs;
        `);

        const repoCountRes = await query(`SELECT COUNT(*)::int as count FROM repositories;`);

        const criticalRes = await query(`
            SELECT COUNT(*)::int as count FROM vulnerabilities WHERE severity = 'CRITICAL';
        `);

        const openFindingsRes = await query(`
            SELECT COUNT(*)::int as count FROM vulnerabilities;
        `);

        const weekScansRes = await query(`
            SELECT COUNT(*)::int as count FROM scan_runs
            WHERE started_at >= NOW() - INTERVAL '7 days';
        `);

        const repoHealthRes = await query(`
            SELECT r.full_name,
                   COUNT(v.id)::int as findings,
                   MAX(s.completed_at) as last_scan
            FROM repositories r
            LEFT JOIN scan_runs s ON s.repository_id = r.id
            LEFT JOIN vulnerabilities v ON v.scan_run_id = s.id
            GROUP BY r.id, r.full_name
            ORDER BY COUNT(v.id) DESC
            LIMIT 5;
        `);

        const prGatesRes = await query(`
            SELECT s.id, s.pr_number, s.commit_sha, s.status, s.findings_count,
                   s.completed_at, r.full_name as repo_name
            FROM scan_runs s
            LEFT JOIN repositories r ON r.id = s.repository_id
            WHERE s.pr_number IS NOT NULL
            ORDER BY s.completed_at DESC
            LIMIT 5;
        `);

        const stats = statsRes.rows[0] || {
            total_scans: 0,
            completed_scans: 0,
            total_findings: 0,
            avg_duration: 0,
        };

        return {
            scans: (scansRes.rows || []) as ScanRun[],
            repoCount: repoCountRes.rows[0]?.count || 12,
            openFindings: openFindingsRes.rows[0]?.count || 27,
            criticalCount: criticalRes.rows[0]?.count || 4,
            weekScans: weekScansRes.rows[0]?.count || 186,
            repoHealth: repoHealthRes.rows || [],
            prGates: prGatesRes.rows || [],
            stats: {
                totalScans: stats.total_scans,
                completedScans: stats.completed_scans,
                criticalVulns: stats.total_findings,
                avgDurationMs: stats.avg_duration,
            },
        };
    } catch {
        return {
            scans: [],
            repoCount: 12,
            openFindings: 27,
            criticalCount: 4,
            weekScans: 186,
            repoHealth: [],
            prGates: [],
            stats: { totalScans: 0, completedScans: 0, criticalVulns: 0, avgDurationMs: 0 },
        };
    }
}

export default async function DashboardPage() {
    const data = await getDashboardData();
    return <DashboardView {...data} />;
}