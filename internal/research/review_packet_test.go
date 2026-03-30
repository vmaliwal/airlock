package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReviewPacket(t *testing.T) {
	tmp := t.TempDir()
	result := FixResult{
		Issue: GitHubIssue{
			Owner:   "owner",
			Repo:    "repo",
			Number:  123,
			Title:   "fix parser edge case",
			HTMLURL: "https://github.com/owner/repo/issues/123",
			Labels:  []string{"bug", "help wanted"},
		},
		RepoPath:            tmp,
		PlanInput:           PlanInput{FailureText: "fix parser edge case", FailingCommand: "go test ./pkg/foo -run TestEdgeCase"},
		ReadonlySummaryPath: filepath.Join(tmp, "readonly-summary.json"),
		Synthesis:           SynthesisReport{Summary: "planner suggested preserving buffered content at EOF"},
		AutofixContractSummary: filepath.Join(tmp, "autofix-summary.json"),
		FixLoop: AutofixLoopSummary{
			WinningAttempt: "attempt-1",
			Rounds: []AutofixLoopRound{{Round: 1, StopReason: "winner_found", Success: true}},
		},
	}
	summary := RunSummary{
		AirlockVersion:          "v0.3.0",
		Backend:                 "lima",
		RepoSHA:                 "abc123",
		ReproStatus:             ReproStatusReproduced,
		ValidationScope:         "target_only",
		FixConfidence:           "medium",
		AttemptCount:            1,
		RoundCount:              1,
		CredibleAdvancement:     true,
		VerifiedIssueResolution: false,
		WinningAttempt:          "attempt-1",
	}
	proof := ProofState{
		ReproStatus:      ReproStatusReproduced,
		ValidationScope:  "target_only",
		FixConfidence:    "medium",
		ConfidenceReason: "reproduced before patch and passed target validation",
	}
	path, err := WriteReviewPacket(result, summary, proof)
	if err != nil {
		t.Fatalf("WriteReviewPacket error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read packet: %v", err)
	}
	text := string(data)
	checks := []string{
		"# Review Packet",
		"issue: [#123 fix parser edge case](https://github.com/owner/repo/issues/123)",
		"inferred failing command: `go test ./pkg/foo -run TestEdgeCase`",
		"repro_status: `reproduced`",
		"planner suggested preserving buffered content at EOF",
		"winning_attempt: `attempt-1`",
		"| Credible advancement | `true` |",
		"## Residual Uncertainty",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("review packet missing %q\n%s", want, text)
		}
	}
}
