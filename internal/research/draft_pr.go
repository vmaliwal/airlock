package research

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteDraftPRArtifact(result FixResult, summary RunSummary, proof ProofState) (string, error) {
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
		dir, err := os.MkdirTemp("", "airlock-draft-pr-")
		if err != nil {
			return "", err
		}
		baseDir = dir
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(baseDir, "draft-pr.md")
	if err := writeText(path, renderDraftPRArtifact(result, summary, proof)); err != nil {
		return "", err
	}
	return path, nil
}

func renderDraftPRArtifact(result FixResult, summary RunSummary, proof ProofState) string {
	var b strings.Builder
	title := strings.TrimSpace(result.Issue.Title)
	if title == "" {
		title = strings.TrimSpace(result.PlanInput.FailureText)
	}
	if title == "" {
		title = fmt.Sprintf("fix issue #%d", result.Issue.Number)
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	fmt.Fprintf(&b, "## Summary\n")
	fmt.Fprintf(&b, "- Issue: %s\n", result.Issue.HTMLURL)
	if cmd := strings.TrimSpace(result.PlanInput.FailingCommand); cmd != "" {
		fmt.Fprintf(&b, "- Reproduction command: `%s`\n", cmd)
	}
	fmt.Fprintf(&b, "- Repro status: `%s`\n", emptyFallback(summary.ReproStatus, proof.ReproStatus, ReproStatusNotReproduced))
	fmt.Fprintf(&b, "- Validation scope: `%s`\n", emptyFallback(summary.ValidationScope, proof.ValidationScope, "reproduction_only"))
	fmt.Fprintf(&b, "- Fix confidence: `%s`\n", emptyFallback(summary.FixConfidence, proof.FixConfidence, "none"))

	fmt.Fprintf(&b, "\n## Rationale\n")
	if strings.TrimSpace(result.Synthesis.Summary) != "" {
		fmt.Fprintf(&b, "%s\n", result.Synthesis.Summary)
	} else if strings.TrimSpace(summary.WinningAttempt) != "" {
		fmt.Fprintf(&b, "This change promotes the winning bounded attempt `%s` from the Airlock fix loop.\n", summary.WinningAttempt)
	} else {
		fmt.Fprintf(&b, "This draft is based on the current Airlock review packet and linked artifacts.\n")
	}

	fmt.Fprintf(&b, "\n## Evidence\n")
	fmt.Fprintf(&b, "- Credible advancement: `%t`\n", summary.CredibleAdvancement)
	fmt.Fprintf(&b, "- Verified issue resolution: `%t`\n", summary.VerifiedIssueResolution)
	if result.ReviewPacketPath != "" {
		fmt.Fprintf(&b, "- Review packet: `%s`\n", result.ReviewPacketPath)
	}
	if result.AutofixContractSummary != "" {
		fmt.Fprintf(&b, "- Autofix summary: `%s`\n", result.AutofixContractSummary)
	}
	if result.ReadonlySummaryPath != "" {
		fmt.Fprintf(&b, "- Readonly summary: `%s`\n", result.ReadonlySummaryPath)
	}

	fmt.Fprintf(&b, "\n## Residual Uncertainty\n")
	if strings.TrimSpace(proof.ConfidenceReason) != "" {
		fmt.Fprintf(&b, "- %s\n", proof.ConfidenceReason)
	}
	if !summary.VerifiedIssueResolution {
		fmt.Fprintf(&b, "- Full issue-resolution proof is not yet complete.\n")
	}
	if strings.TrimSpace(summary.FailureCategory) != "" {
		fmt.Fprintf(&b, "- Run stop/failure category: `%s`.\n", summary.FailureCategory)
	}
	return b.String()
}
