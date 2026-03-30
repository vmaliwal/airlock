package research

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteReviewPacket(result FixResult, summary RunSummary, proof ProofState) (string, error) {
	baseDir := ""
	switch {
	case strings.TrimSpace(result.AutofixContractSummary) != "":
		baseDir = filepath.Dir(result.AutofixContractSummary)
	case strings.TrimSpace(result.ReadonlySummaryPath) != "":
		baseDir = filepath.Dir(result.ReadonlySummaryPath)
	default:
		baseDir = result.RepoPath
	}
	if strings.TrimSpace(baseDir) == "" {
		dir, err := os.MkdirTemp("", "airlock-review-packet-")
		if err != nil {
			return "", err
		}
		baseDir = dir
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(baseDir, "review-packet.md")
	if err := writeText(path, renderReviewPacket(result, summary, proof)); err != nil {
		return "", err
	}
	return path, nil
}

func renderReviewPacket(result FixResult, summary RunSummary, proof ProofState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Review Packet\n\n")
	fmt.Fprintf(&b, "## Issue\n")
	fmt.Fprintf(&b, "- repo: `%s/%s`\n", result.Issue.Owner, result.Issue.Repo)
	fmt.Fprintf(&b, "- issue: [%s](%s)\n", packetIssueTitle(result.Issue), result.Issue.HTMLURL)
	if strings.TrimSpace(result.PlanInput.FailureText) != "" {
		fmt.Fprintf(&b, "- summary: %s\n", result.PlanInput.FailureText)
	}
	if len(result.Issue.Labels) > 0 {
		fmt.Fprintf(&b, "- labels: %s\n", strings.Join(result.Issue.Labels, ", "))
	}
	if cmd := strings.TrimSpace(result.PlanInput.FailingCommand); cmd != "" {
		fmt.Fprintf(&b, "- inferred failing command: `%s`\n", cmd)
	}

	fmt.Fprintf(&b, "\n## Reproduction\n")
	fmt.Fprintf(&b, "- repro_status: `%s`\n", emptyFallback(summary.ReproStatus, proof.ReproStatus, ReproStatusNotReproduced))
	fmt.Fprintf(&b, "- validation_scope: `%s`\n", emptyFallback(summary.ValidationScope, proof.ValidationScope, "reproduction_only"))
	fmt.Fprintf(&b, "- fix_confidence: `%s`\n", emptyFallback(summary.FixConfidence, proof.FixConfidence, "none"))
	if strings.TrimSpace(proof.ConfidenceReason) != "" {
		fmt.Fprintf(&b, "- confidence_reason: %s\n", proof.ConfidenceReason)
	}
	if result.ReadonlySummaryPath != "" {
		fmt.Fprintf(&b, "- readonly_summary: `%s`\n", result.ReadonlySummaryPath)
	}

	fmt.Fprintf(&b, "\n## Fix Loop\n")
	fmt.Fprintf(&b, "- attempts_generated: %d\n", len(result.Synthesis.Attempts))
	fmt.Fprintf(&b, "- rounds: %d\n", packetRoundCount(result.FixLoop, summary))
	fmt.Fprintf(&b, "- credible_advancement: `%t`\n", summary.CredibleAdvancement)
	fmt.Fprintf(&b, "- verified_issue_resolution: `%t`\n", summary.VerifiedIssueResolution)
	if strings.TrimSpace(summary.FailureCategory) != "" {
		fmt.Fprintf(&b, "- failure_category: `%s`\n", summary.FailureCategory)
	}
	if strings.TrimSpace(summary.WinningAttempt) != "" {
		fmt.Fprintf(&b, "- winning_attempt: `%s`\n", summary.WinningAttempt)
	} else if strings.TrimSpace(result.FixLoop.WinningAttempt) != "" {
		fmt.Fprintf(&b, "- winning_attempt: `%s`\n", result.FixLoop.WinningAttempt)
	}
	if strings.TrimSpace(result.AutofixContractSummary) != "" {
		fmt.Fprintf(&b, "- autofix_summary: `%s`\n", result.AutofixContractSummary)
	}

	fmt.Fprintf(&b, "\n## Root Cause / Fix Rationale\n")
	if strings.TrimSpace(result.Synthesis.Summary) != "" {
		fmt.Fprintf(&b, "%s\n", result.Synthesis.Summary)
	} else if strings.TrimSpace(summary.WinningAttempt) != "" {
		fmt.Fprintf(&b, "Winning attempt `%s` validated in the current bounded fix loop.\n", summary.WinningAttempt)
	} else {
		fmt.Fprintf(&b, "No maintainer-grade root-cause summary was synthesized yet; use the linked artifacts below for exact evidence.\n")
	}

	fmt.Fprintf(&b, "\n## Evidence\n")
	fmt.Fprintf(&b, "| Signal | Value |\n")
	fmt.Fprintf(&b, "|---|---|\n")
	fmt.Fprintf(&b, "| Airlock version | `%s` |\n", summary.AirlockVersion)
	fmt.Fprintf(&b, "| Backend | `%s` |\n", emptyFallback(summary.Backend, "unknown"))
	fmt.Fprintf(&b, "| Repo SHA | `%s` |\n", emptyFallback(summary.RepoSHA, "unknown"))
	fmt.Fprintf(&b, "| Repro status | `%s` |\n", emptyFallback(summary.ReproStatus, proof.ReproStatus, ReproStatusNotReproduced))
	fmt.Fprintf(&b, "| Validation scope | `%s` |\n", emptyFallback(summary.ValidationScope, proof.ValidationScope, "reproduction_only"))
	fmt.Fprintf(&b, "| Fix confidence | `%s` |\n", emptyFallback(summary.FixConfidence, proof.FixConfidence, "none"))
	fmt.Fprintf(&b, "| Attempts | `%d` |\n", summary.AttemptCount)
	fmt.Fprintf(&b, "| Rounds | `%d` |\n", packetRoundCount(result.FixLoop, summary))
	fmt.Fprintf(&b, "| Credible advancement | `%t` |\n", summary.CredibleAdvancement)
	fmt.Fprintf(&b, "| Verified issue resolution | `%t` |\n", summary.VerifiedIssueResolution)

	fmt.Fprintf(&b, "\n## Residual Uncertainty\n")
	switch {
	case !summary.CredibleAdvancement:
		fmt.Fprintf(&b, "- The current run does not meet Airlock's threshold for credible advancement.\n")
	case !summary.VerifiedIssueResolution:
		fmt.Fprintf(&b, "- The fix may be promising, but full issue-resolution proof remains incomplete.\n")
	default:
		fmt.Fprintf(&b, "- Remaining uncertainty appears bounded by the recorded validation scope.\n")
	}
	if strings.TrimSpace(summary.FailureCategory) != "" {
		fmt.Fprintf(&b, "- Failure/stop category: `%s`.\n", summary.FailureCategory)
	}
	if len(result.FixLoop.Rounds) > 0 {
		last := result.FixLoop.Rounds[len(result.FixLoop.Rounds)-1]
		if strings.TrimSpace(last.StopReason) != "" {
			fmt.Fprintf(&b, "- Final loop stop reason: `%s`.\n", last.StopReason)
		}
	}

	fmt.Fprintf(&b, "\n## Artifact Paths\n")
	if result.ReadonlySummaryPath != "" {
		fmt.Fprintf(&b, "- readonly summary: `%s`\n", result.ReadonlySummaryPath)
	}
	if result.AutofixContractSummary != "" {
		fmt.Fprintf(&b, "- autofix summary: `%s`\n", result.AutofixContractSummary)
	}
	fmt.Fprintf(&b, "\n")
	return b.String()
}

func packetRoundCount(loop AutofixLoopSummary, summary RunSummary) int {
	if len(loop.Rounds) > 0 {
		return len(loop.Rounds)
	}
	if summary.RoundCount > 0 {
		return summary.RoundCount
	}
	return 0
}

func packetIssueTitle(issue GitHubIssue) string {
	title := strings.TrimSpace(issue.Title)
	if title != "" {
		return fmt.Sprintf("#%d %s", issue.Number, title)
	}
	return fmt.Sprintf("#%d", issue.Number)
}

func emptyFallback(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
