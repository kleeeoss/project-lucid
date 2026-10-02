package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMockReportsAreDeterministicAndValid(t *testing.T) {
	cases := []VulnerabilityReport{MockSQLInjectionReport(), MockCommandInjectionReport()}
	for _, report := range cases {
		if report.ScanID == "" || report.FilePath == "" || report.TargetFile == "" || report.Language == "" {
			t.Fatalf("mock report missing required top-level fields: %+v", report)
		}
		if report.FilePath != report.TargetFile {
			t.Fatalf("mock report should set TargetFile alias equal to FilePath")
		}
		if len(report.Vulnerabilities) != 1 {
			t.Fatalf("expected exactly one vulnerability, got %d", len(report.Vulnerabilities))
		}
		v := report.Vulnerabilities[0]
		if v.ID == "" || v.FilePath != report.FilePath || v.RuleID == "" || v.CWE == "" || len(v.TaintPath) == 0 {
			t.Fatalf("mock vulnerability invalid: %+v", v)
		}
		if v.ConfidenceScore < 0 || v.ConfidenceScore > 1 {
			t.Fatalf("confidence out of range: %f", v.ConfidenceScore)
		}
		if _, err := json.Marshal(report); err != nil {
			t.Fatalf("mock report must serialize: %v", err)
		}
	}

	if !reflect.DeepEqual(MockSQLInjectionReport(), MockSQLInjectionReport()) {
		t.Fatalf("SQLi mock report must be deterministic")
	}
	if !reflect.DeepEqual(MockCommandInjectionReport(), MockCommandInjectionReport()) {
		t.Fatalf("command injection mock report must be deterministic")
	}
}
