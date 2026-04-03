// Package notify provides post-run notification integrations.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TestPayload is the JSON body sent after a test run.
type TestPayload struct {
	Tool      string  `json:"tool"`
	Timestamp string  `json:"timestamp"`
	Total     int     `json:"total"`
	Passed    int     `json:"passed"`
	Failed    int     `json:"failed"`
	Skipped   int     `json:"skipped"`
	PassPct   float64 `json:"pass_pct"`
}

// CovPayload is the JSON body sent after a coverage run.
type CovPayload struct {
	Tool       string  `json:"tool"`
	Timestamp  string  `json:"timestamp"`
	LinesPct   float64 `json:"lines_pct"`
	LinesCov   int     `json:"lines_covered"`
	LinesTotal int     `json:"lines_total"`
	BranchPct  float64 `json:"branch_pct,omitempty"`
	FuncPct    float64 `json:"func_pct,omitempty"`
	FileCount  int     `json:"file_count"`
}

// PostTest sends test run statistics to webhookURL via HTTP POST.
func PostTest(webhookURL string, total, passed, failed, skipped int) error {
	pct := 0.0
	if total > 0 {
		pct = float64(passed) / float64(total) * 100
	}
	payload := TestPayload{
		Tool:      "trep",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Total:     total,
		Passed:    passed,
		Failed:    failed,
		Skipped:   skipped,
		PassPct:   pct,
	}
	return post(webhookURL, payload)
}

// PostCov sends coverage statistics to webhookURL via HTTP POST.
func PostCov(webhookURL string, linesPct float64, linesCov, linesTotal int, branchPct, funcPct float64, fileCount int) error {
	payload := CovPayload{
		Tool:       "trep",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		LinesPct:   linesPct,
		LinesCov:   linesCov,
		LinesTotal: linesTotal,
		BranchPct:  branchPct,
		FuncPct:    funcPct,
		FileCount:  fileCount,
	}
	return post(webhookURL, payload)
}

func post(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook marshal: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
