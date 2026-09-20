package rules

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"lucid-ci/engine/graph"
	"lucid-ci/engine/models"
)

const (
	InsecureSecretsRuleID   = "LUCID-SEC-003"
	InsecureSecretsRuleName = "Insecure Hardcoded Secret"
	InsecureSecretsCWE      = "CWE-798"
	InsecureSecretsOWASP    = "A07:2021-Identification and Authentication Failures"
	secretEntropyThreshold  = 4.5
)

var (
	awsAccessKeyRE = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	githubPATRE    = regexp.MustCompile(`ghp_[A-Za-z0-9_]{36}`)
	privateKeyRE   = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	quotedValueRE  = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"|'([^'\\]*(?:\\.[^'\\]*)*)'`)
	secretNameRE   = regexp.MustCompile(`(?i)(secret|token|api[_-]?key|password|passwd|credential|private[_-]?key)`)
)

func DetectInsecureSecrets(filePath string, source []byte) ([]models.Vulnerability, error) {
	g, err := graph.NormalizeSource(filePath, source)
	if err != nil {
		return nil, err
	}
	return DetectInsecureSecretsInGraph(g), nil
}

func DetectInsecureSecretsInGraph(g *graph.Graph) []models.Vulnerability {
	if g == nil {
		return nil
	}
	vulns := make([]models.Vulnerability, 0)
	seen := map[string]bool{}
	accepted := make([]graph.Node, 0)
	nodes := append([]graph.Node(nil), g.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		spanI := nodes[i].ByteEnd - nodes[i].ByteStart
		spanJ := nodes[j].ByteEnd - nodes[j].ByteStart
		if spanI == spanJ {
			return nodes[i].ID < nodes[j].ID
		}
		return spanI < spanJ
	})
	for _, node := range nodes {
		if node.Type != graph.SemanticVariableDeclaration && node.Type != graph.SemanticAssignmentExpression {
			continue
		}
		if containsAcceptedSecret(node, accepted) {
			continue
		}
		secret, ok := secretCandidate(node)
		if !ok {
			continue
		}
		id := deterministicVulnerabilityID(InsecureSecretsRuleID, node)
		if seen[id] {
			continue
		}
		seen[id] = true
		accepted = append(accepted, node)
		vulns = append(vulns, models.Vulnerability{
			ID:              id,
			FilePath:        g.FilePath,
			RuleID:          InsecureSecretsRuleID,
			RuleName:        InsecureSecretsRuleName,
			CWE:             InsecureSecretsCWE,
			OWASP:           InsecureSecretsOWASP,
			Severity:        models.SeverityHigh,
			ConfidenceScore: secret.confidence,
			Description:     "A high-entropy or well-known secret pattern is hardcoded in source code.",
			SourceNode:      astRef(node),
			SinkNode:        astRef(node),
			TaintPath:       []models.ASTNodeRef{astRef(node)},
			VulnerableCode:  node.CodeSnippet,
			LineStart:       node.LineStart,
			LineEnd:         node.LineEnd,
			ColStart:        node.ColumnStart,
			ColEnd:          node.ColumnEnd,
		})
	}
	return vulns
}

func containsAcceptedSecret(node graph.Node, accepted []graph.Node) bool {
	for _, prior := range accepted {
		if prior.ByteStart >= node.ByteStart && prior.ByteEnd <= node.ByteEnd {
			return true
		}
	}
	return false
}

type secretMatch struct{ confidence float64 }

func secretCandidate(node graph.Node) (secretMatch, bool) {
	code := node.CodeSnippet
	if awsAccessKeyRE.MatchString(code) || githubPATRE.MatchString(code) || privateKeyRE.MatchString(code) {
		return secretMatch{confidence: 0.99}, true
	}
	if !secretNameRE.MatchString(node.Name) && !secretNameRE.MatchString(code) {
		return secretMatch{}, false
	}
	for _, value := range quotedValues(code) {
		if len(value) < 16 {
			continue
		}
		entropy := ShannonEntropy(value)
		if entropy > secretEntropyThreshold {
			confidence := 0.75 + math.Min((entropy-secretEntropyThreshold)/2.0, 0.24)
			return secretMatch{confidence: confidence}, true
		}
	}
	return secretMatch{}, false
}

func quotedValues(code string) []string {
	matches := quotedValueRE.FindAllStringSubmatch(code, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			values = append(values, match[1])
			continue
		}
		if len(match) > 2 && match[2] != "" {
			values = append(values, match[2])
		}
	}
	return values
}

func ShannonEntropy(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	counts := map[rune]float64{}
	for _, r := range value {
		counts[r]++
	}
	length := float64(len([]rune(value)))
	entropy := 0.0
	for _, count := range counts {
		p := count / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}
