export type ScanStatus = "QUEUED" | "SCANNING" | "ANALYZING_AI" | "SANDBOXING" | "COMPLETED" | "FAILED";
export type VulnSeverity = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW" | "INFO";

export interface Repository {
    id: number;
    full_name: string;
    installation_id: number;
    default_branch: string;
    created_at: string;
}

export interface ScanRun {
    id: string;
    repository_id: number;
    repo_name?: string;
    pr_number: number;
    commit_sha: string;
    status: ScanStatus;
    check_run_id?: number | null;
    findings_count: number;
    scan_duration_ms?: number | null;
    started_at: string;
    completed_at?: string | null;
}

export interface Vulnerability {
    id: string;
    scan_run_id: string;
    rule_id: string;
    cwe: string;
    severity: VulnSeverity;
    confidence_score: number;
    file_path: string;
    line_start: number;
    line_end: number;
    vulnerable_code: string;
    ai_remediation_patch?: string | null;
    ai_explanation?: string | null;
    sandbox_verified: boolean;
    ast_graph_json?: any;
    created_at: string;
}

export interface DashboardMetrics {
    totalScans: number;
    criticalVulns: number;
    completedScans: number;
    avgDurationMs: number;
}