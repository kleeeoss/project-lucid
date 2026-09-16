package models

import (
	"time"
)

// ScanStatus represents the lifecycle state of a scan run.
type ScanStatus string

const (
	ScanStatusQueued      ScanStatus = "QUEUED"
	ScanStatusScanning    ScanStatus = "SCANNING"
	ScanStatusAnalyzingAI ScanStatus = "ANALYZING_AI"
	ScanStatusSandboxing  ScanStatus = "SANDBOXING"
	ScanStatusCompleted   ScanStatus = "COMPLETED"
	ScanStatusFailed      ScanStatus = "FAILED"
)

// VulnSeverity represents the severity rating of a vulnerability.
type VulnSeverity string

const (
	SeverityCritical VulnSeverity = "CRITICAL"
	SeverityHigh     VulnSeverity = "HIGH"
	SeverityMedium   VulnSeverity = "MEDIUM"
	SeverityLow      VulnSeverity = "LOW"
	SeverityInfo     VulnSeverity = "INFO"
)

// Repository represents a row in the 'repositories' table.
type Repository struct {
	ID             int64     `json:"id"`
	FullName       string    `json:"full_name"`
	InstallationID int64     `json:"installation_id"`
	DefaultBranch  string    `json:"default_branch"`
	CreatedAt      time.Time `json:"created_at"`
}

// ScanRun represents a row in the 'scan_runs' table.
type ScanRun struct {
	ID             string     `json:"id"`
	RepositoryID   int64      `json:"repository_id"`
	PRNumber       int        `json:"pr_number"`
	CommitSHA      string     `json:"commit_sha"`
	Status         ScanStatus `json:"status"`
	CheckRunID     *int64     `json:"check_run_id,omitempty"`
	FindingsCount  int        `json:"findings_count"`
	ScanDurationMs *int       `json:"scan_duration_ms,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// Vulnerability represents a row in the 'vulnerabilities' table (CTR-008).
type Vulnerability struct {
	ID                 string       `json:"id"`
	ScanRunID          string       `json:"scan_run_id"`
	RuleID             string       `json:"rule_id"`
	CWE                string       `json:"cwe"`
	Severity           VulnSeverity `json:"severity"`
	ConfidenceScore    float64      `json:"confidence_score"`
	FilePath           string       `json:"file_path"`
	LineStart          int          `json:"line_start"`
	LineEnd            int          `json:"line_end"`
	VulnerableCode     string       `json:"vulnerable_code"`
	AIRemediationPatch *string      `json:"ai_remediation_patch,omitempty"`
	AIExplanation      *string      `json:"ai_explanation,omitempty"`
	SandboxVerified    bool         `json:"sandbox_verified"`
	ASTGraphJSON       []byte       `json:"ast_graph_json,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
}
