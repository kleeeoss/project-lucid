// src/app/repositories/page.tsx
import { query } from "@/lib/db";
import { RepositoriesView, RepositoryItem } from "@/components/RepositoriesView";

export const revalidate = 5;

async function getRepositories(): Promise<RepositoryItem[]> {
    try {
        const res = await query(`
            SELECT r.id, r.full_name, r.default_branch, r.created_at,
                   COUNT(DISTINCT s.id)::int as scan_count,
                   COALESCE(SUM(s.findings_count), 0)::int as total_findings,
                   MAX(s.completed_at) as last_scan_at
            FROM repositories r
            LEFT JOIN scan_runs s ON s.repository_id = r.id
            GROUP BY r.id, r.full_name, r.default_branch, r.created_at
            ORDER BY r.created_at DESC;
        `);
        return res.rows as RepositoryItem[];
    } catch (error) {
        console.error("Failed to query repositories:", error);
        return [];
    }
}

export default async function RepositoriesPage() {
    const repos = await getRepositories();
    return <RepositoriesView repos={repos} />;
}
