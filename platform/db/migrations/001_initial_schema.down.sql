-- Down Migration: 001_initial_schema.down.sql

DROP TABLE IF EXISTS vulnerabilities CASCADE;
DROP TABLE IF EXISTS scan_runs CASCADE;
DROP TABLE IF EXISTS repositories CASCADE;

DROP TYPE IF EXISTS vuln_severity;
DROP TYPE IF EXISTS scan_status;