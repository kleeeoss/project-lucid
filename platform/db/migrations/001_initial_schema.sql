-- Migration: 001_initial_schema.sql (Contract CTR-008)
-- Description: Lucid-CI 3-table baseline relational schema for repositories, scan runs, and vulnerabilities.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enum types for scan lifecycle states and vulnerability severities
DO $$ BEGIN
    CREATE TYPE scan_status AS ENUM ('QUEUED', 'SCANNING', 'ANALYZING_AI', 'SANDBOXING', 'COMPLETED', 'FAILED');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE vuln_severity AS ENUM ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Repositories: Monitored GitHub repositories where Lucid-CI is installed
CREATE TABLE IF NOT EXISTS repositories (
    id              BIGINT PRIMARY KEY,              -- GitHub Repository ID
    full_name       VARCHAR(255) NOT NULL UNIQUE,   -- "owner/repo"
    installation_id BIGINT NOT NULL,                -- GitHub App Installation ID
    default_branch  VARCHAR(100) NOT NULL DEFAULT 'main',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Scan Runs: One record per scan trigger (Pull Request event)
CREATE TABLE IF NOT EXISTS scan_runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id       BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pr_number           INT NOT NULL,
    commit_sha          VARCHAR(40) NOT NULL,
    status              scan_status NOT NULL DEFAULT 'QUEUED',
    check_run_id        BIGINT,                     -- GitHub Check Run ID for live annotation updates
    findings_count      INT NOT NULL DEFAULT 0,
    scan_duration_ms    INT,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_scan_runs_repo_pr ON scan_runs(repository_id, pr_number);
CREATE INDEX IF NOT EXISTS idx_scan_runs_status ON scan_runs(status);

-- Vulnerabilities: Individual findings identified during a scan
CREATE TABLE IF NOT EXISTS vulnerabilities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_run_id         UUID NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    rule_id             VARCHAR(50) NOT NULL,       -- e.g., 'LUCID-SEC-001'
    cwe                 VARCHAR(20) NOT NULL,       -- e.g., 'CWE-89'
    severity            vuln_severity NOT NULL,
    confidence_score    NUMERIC(3,2) NOT NULL,      -- 0.00 - 1.00
    file_path           VARCHAR(500) NOT NULL,
    line_start          INT NOT NULL,
    line_end            INT NOT NULL,
    vulnerable_code     TEXT NOT NULL,
    ai_remediation_patch TEXT,                      -- Suggested patch replacement from SLM
    ai_explanation      TEXT,                       -- Architectural explanation
    sandbox_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    ast_graph_json      JSONB,                      -- React Flow nodes and edges (CTR-009)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vulns_scan_run ON vulnerabilities(scan_run_id);
CREATE INDEX IF NOT EXISTS idx_vulns_severity ON vulnerabilities(severity);