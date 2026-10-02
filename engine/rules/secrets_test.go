package rules

import "testing"

func TestDetectInsecureSecretsAWSKey(t *testing.T) {
	vulns, err := DetectInsecureSecrets("config.js", []byte(`const awsKey = "AKIA1234567890ABCDEF";`))
	if err != nil {
		t.Fatalf("DetectInsecureSecrets: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected AWS key finding, got %+v", vulns)
	}
	if vulns[0].RuleID != InsecureSecretsRuleID || vulns[0].CWE != InsecureSecretsCWE {
		t.Fatalf("unexpected metadata: %+v", vulns[0])
	}
}

func TestDetectInsecureSecretsHighEntropyToken(t *testing.T) {
	vulns, err := DetectInsecureSecrets("settings.py", []byte(`api_token = "z9Aq7LmN4pR8xT2vB6yC0dE3fG5hJ7kL"`))
	if err != nil {
		t.Fatalf("DetectInsecureSecrets: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected high entropy token finding, got %+v", vulns)
	}
	if vulns[0].ConfidenceScore <= 0.75 || vulns[0].ConfidenceScore > 1 {
		t.Fatalf("unexpected confidence: %f", vulns[0].ConfidenceScore)
	}
}

func TestDetectInsecureSecretsPrivateKey(t *testing.T) {
	source := []byte(`const privateKey = "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----";`)
	vulns, err := DetectInsecureSecrets("config.js", source)
	if err != nil {
		t.Fatalf("DetectInsecureSecrets: %v", err)
	}
	if len(vulns) != 1 {
		t.Fatalf("expected private key finding, got %+v", vulns)
	}
}

func TestDetectInsecureSecretsCleanLowEntropy(t *testing.T) {
	vulns, err := DetectInsecureSecrets("config.js", []byte(`const password = "development"; const label = "AKIA-not-real";`))
	if err != nil {
		t.Fatalf("DetectInsecureSecrets: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("low entropy values should not produce findings: %+v", vulns)
	}
}

func TestShannonEntropy(t *testing.T) {
	if ShannonEntropy("aaaaaaaaaaaaaaaa") != 0 {
		t.Fatalf("repeated character entropy should be zero")
	}
	if ShannonEntropy("abcdefghijklmnopqrstuvwxyz012345") <= 4.5 {
		t.Fatalf("varied token should exceed threshold")
	}
}
