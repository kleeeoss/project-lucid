package engine

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"lucid-platform/db"
	"lucid-platform/models"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// fetchPRDiff downloads the actual raw code changes from GitHub
func fetchPRDiff(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func RunScan(payload models.WebhookPayload) {
	sha := payload.PullRequest.Head.SHA
	diffURL := payload.PullRequest.DiffURL

	fmt.Printf("\n🧠 [AI Worker] Starting Gemini Deep Scan on Commit: %s...\n", sha[:7])
	fmt.Printf("📥 [AI Worker] Downloading real code from: %s\n", diffURL)

	// 1. Fetch the REAL code from GitHub!
	prCode, err := fetchPRDiff(diffURL)
	if err != nil || prCode == "" {
		log.Printf("❌ [Error] Failed to fetch PR diff: %v\n", err)
		db.UpdateScanStatus(sha, "error_fetch_failed")
		return
	}

	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("❌ [Error] GEMINI_API_KEY not found!")
		db.UpdateScanStatus(sha, "error_no_key")
		return
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("❌ [Error] Failed to connect to Gemini: %v\n", err)
		db.UpdateScanStatus(sha, "error_connection")
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	// 2. Feed the REAL code to the AI
	prompt := fmt.Sprintf(`
	You are a strict Senior Security Engineer.
	Review this code snippet for vulnerabilities.
	If it has critical vulnerabilities, reply with EXACTLY "FAILED" on the first line, followed by the reason on the next line.
	If it is perfectly safe, reply with EXACTLY "PASSED" on the first line.

	Code to review:
	%s
	`, prCode)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("❌ [Error] AI Scan failed: %v\n", err)
		db.UpdateScanStatus(sha, "error_scan_failed")
		return
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		aiResponse := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
		lines := strings.Split(strings.TrimSpace(aiResponse), "\n")
		finalStatus := strings.ToLower(strings.TrimSpace(lines[0]))

		fmt.Printf("🤖 [AI Verdict]: %s\n", strings.ToUpper(finalStatus))
		if len(lines) > 1 {
			fmt.Printf("📝 [AI Reason]: %s\n", strings.Join(lines[1:], "\n"))
		}

		db.UpdateScanStatus(sha, finalStatus)
	}
}
