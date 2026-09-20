package models

import "time"

type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
	SeverityInfo     SeverityLevel = "INFO"
)

type VulnerabilityReport struct {
	ScanID          string          `json:"scan_id"`
	FilePath        string          `json:"file_path"`
	TargetFile      string          `json:"target_file,omitempty"`
	Language        string          `json:"language"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	ScanDurationMs  int64           `json:"scan_duration_ms"`
	Timestamp       time.Time       `json:"timestamp"`
}

type Vulnerability struct {
	ID              string        `json:"id"`
	FilePath        string        `json:"file_path"`
	RuleID          string        `json:"rule_id"`
	RuleName        string        `json:"rule_name"`
	CWE             string        `json:"cwe"`
	OWASP           string        `json:"owasp"`
	Severity        SeverityLevel `json:"severity"`
	ConfidenceScore float64       `json:"confidence_score"`
	Description     string        `json:"description"`
	SourceNode      ASTNodeRef    `json:"source_node"`
	SinkNode        ASTNodeRef    `json:"sink_node"`
	TaintPath       []ASTNodeRef  `json:"taint_path"`
	VulnerableCode  string        `json:"vulnerable_code"`
	LineStart       int           `json:"line_start"`
	LineEnd         int           `json:"line_end"`
	ColStart        int           `json:"col_start"`
	ColEnd          int           `json:"col_end"`
}

type ASTNodeRef struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	ByteStart uint32 `json:"byte_start"`
	ByteEnd   uint32 `json:"byte_end"`
}
