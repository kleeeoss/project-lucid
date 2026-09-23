package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lucid-ci/platform/models"
)

// Store defines all persistence operations required by the Platform worker and API.
type Store interface {
	Close()
	Ping(ctx context.Context) error

	UpsertRepository(ctx context.Context, repo *models.Repository) error
	GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error)

	CreateScanRun(ctx context.Context, scan *models.ScanRun) error
	UpdateScanRunStatus(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error
	UpdateCheckRunID(ctx context.Context, scanID string, checkRunID int64) error
	GetScanRunByID(ctx context.Context, scanID string) (*models.ScanRun, error)

	InsertVulnerabilities(ctx context.Context, scanID string, vulns []models.Vulnerability) error
	GetVulnerabilitiesByScanRunID(ctx context.Context, scanID string) ([]models.Vulnerability, error)
}

type pgStore struct {
	pool *pgxpool.Pool
}

// NewStore initializes a new pgx connection pool with sensible production defaults.
func NewStore(ctx context.Context, databaseURL string) (Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &pgStore{pool: pool}, nil
}

func (s *pgStore) Close() {
	s.pool.Close()
}

func (s *pgStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *pgStore) UpsertRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT INTO repositories (id, full_name, installation_id, default_branch, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE
		SET full_name = EXCLUDED.full_name,
		    installation_id = EXCLUDED.installation_id,
		    default_branch = EXCLUDED.default_branch;
	`
	_, err := s.pool.Exec(ctx, query, repo.ID, repo.FullName, repo.InstallationID, repo.DefaultBranch)
	if err != nil {
		return fmt.Errorf("UpsertRepository: %w", err)
	}
	return nil
}

func (s *pgStore) GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error) {
	query := `
		SELECT id, full_name, installation_id, default_branch, created_at
		FROM repositories
		WHERE id = $1;
	`
	var repo models.Repository
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.InstallationID,
		&repo.DefaultBranch,
		&repo.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetRepositoryByID: %w", err)
	}
	return &repo, nil
}

func (s *pgStore) CreateScanRun(ctx context.Context, scan *models.ScanRun) error {
	query := `
		INSERT INTO scan_runs (id, repository_id, pr_number, commit_sha, status, findings_count, started_at)
		VALUES (COALESCE(NULLIF($6, '')::uuid, gen_random_uuid()), $1, $2, $3, $4::scan_status, $5, NOW())
		RETURNING id, started_at;
	`
	err := s.pool.QueryRow(ctx, query,
		scan.RepositoryID,
		scan.PRNumber,
		scan.CommitSHA,
		string(scan.Status),
		scan.FindingsCount,
		scan.ID,
	).Scan(&scan.ID, &scan.StartedAt)
	if err != nil {
		return fmt.Errorf("CreateScanRun: %w", err)
	}
	return nil
}

func (s *pgStore) UpdateScanRunStatus(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error {
	query := `
		UPDATE scan_runs
		SET status = $1::scan_status,
		    findings_count = $2,
		    scan_duration_ms = $3,
		    completed_at = CASE WHEN $1::text IN ('COMPLETED', 'FAILED') THEN NOW() ELSE completed_at END
		WHERE id = $4::uuid;
	`
	tag, err := s.pool.Exec(ctx, query, string(status), findingsCount, durationMs, scanID)
	if err != nil {
		return fmt.Errorf("UpdateScanRunStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("UpdateScanRunStatus: scan_run %s not found", scanID)
	}
	return nil
}

func (s *pgStore) UpdateCheckRunID(ctx context.Context, scanID string, checkRunID int64) error {
	query := `
		UPDATE scan_runs
		SET check_run_id = $1
		WHERE id = $2::uuid;
	`
	_, err := s.pool.Exec(ctx, query, checkRunID, scanID)
	if err != nil {
		return fmt.Errorf("UpdateCheckRunID: %w", err)
	}
	return nil
}

func (s *pgStore) GetScanRunByID(ctx context.Context, scanID string) (*models.ScanRun, error) {
	query := `
		SELECT id, repository_id, pr_number, commit_sha, status, check_run_id,
		       findings_count, scan_duration_ms, started_at, completed_at
		FROM scan_runs
		WHERE id = $1::uuid;
	`
	var scan models.ScanRun
	err := s.pool.QueryRow(ctx, query, scanID).Scan(
		&scan.ID,
		&scan.RepositoryID,
		&scan.PRNumber,
		&scan.CommitSHA,
		&scan.Status,
		&scan.CheckRunID,
		&scan.FindingsCount,
		&scan.ScanDurationMs,
		&scan.StartedAt,
		&scan.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetScanRunByID: %w", err)
	}
	return &scan, nil
}

func (s *pgStore) InsertVulnerabilities(ctx context.Context, scanID string, vulns []models.Vulnerability) error {
	if len(vulns) == 0 {
		return nil
	}

	rows := make([][]any, len(vulns))
	for i, v := range vulns {
		rows[i] = []any{
			scanID,
			v.RuleID,
			v.CWE,
			v.Severity,
			v.ConfidenceScore,
			v.FilePath,
			v.LineStart,
			v.LineEnd,
			v.VulnerableCode,
			v.AIRemediationPatch,
			v.AIExplanation,
			v.SandboxVerified,
			v.ASTGraphJSON,
		}
	}

	columns := []string{
		"scan_run_id",
		"rule_id",
		"cwe",
		"severity",
		"confidence_score",
		"file_path",
		"line_start",
		"line_end",
		"vulnerable_code",
		"ai_remediation_patch",
		"ai_explanation",
		"sandbox_verified",
		"ast_graph_json",
	}

	copyCount, err := s.pool.CopyFrom(
		ctx,
		pgx.Identifier{"vulnerabilities"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("InsertVulnerabilities batch insert: %w", err)
	}

	if int(copyCount) != len(vulns) {
		return fmt.Errorf("InsertVulnerabilities: expected %d rows copied, got %d", len(vulns), copyCount)
	}

	return nil
}

func (s *pgStore) GetVulnerabilitiesByScanRunID(ctx context.Context, scanID string) ([]models.Vulnerability, error) {
	query := `
		SELECT id, scan_run_id, rule_id, cwe, severity, confidence_score,
		       file_path, line_start, line_end, vulnerable_code,
		       ai_remediation_patch, ai_explanation, sandbox_verified,
		       ast_graph_json, created_at
		FROM vulnerabilities
		WHERE scan_run_id = $1::uuid
		ORDER BY line_start ASC;
	`
	rows, err := s.pool.Query(ctx, query, scanID)
	if err != nil {
		return nil, fmt.Errorf("GetVulnerabilitiesByScanRunID: %w", err)
	}
	defer rows.Close()

	var vulns []models.Vulnerability
	for rows.Next() {
		var v models.Vulnerability
		if err := rows.Scan(
			&v.ID,
			&v.ScanRunID,
			&v.RuleID,
			&v.CWE,
			&v.Severity,
			&v.ConfidenceScore,
			&v.FilePath,
			&v.LineStart,
			&v.LineEnd,
			&v.VulnerableCode,
			&v.AIRemediationPatch,
			&v.AIExplanation,
			&v.SandboxVerified,
			&v.ASTGraphJSON,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("GetVulnerabilitiesByScanRunID scan row: %w", err)
		}
		vulns = append(vulns, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetVulnerabilitiesByScanRunID rows error: %w", err)
	}

	return vulns, nil
}
