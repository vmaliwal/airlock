package research

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AutofixLoopPolicy struct {
	MaxRounds int `json:"max_rounds,omitempty"`
}

type AutofixLoopRound struct {
	Round                 int      `json:"round"`
	NewAttemptCount       int      `json:"new_attempt_count"`
	DuplicateAttemptCount int      `json:"duplicate_attempt_count"`
	AttemptNames          []string `json:"attempt_names"`
	SummaryPath           string   `json:"summary_path,omitempty"`
	Success               bool     `json:"success"`
	WinningAttempt        string   `json:"winning_attempt,omitempty"`
	WinnerPromoted        bool     `json:"winner_promoted,omitempty"`
	PromotedCheckpoint    string   `json:"promoted_checkpoint,omitempty"`
	StopReason            string   `json:"stop_reason,omitempty"`
}

type AutofixLoopSummary struct {
	Objective        string             `json:"objective"`
	Repo             string             `json:"repo"`
	Success          bool               `json:"success"`
	WinningAttempt   string             `json:"winningAttempt,omitempty"`
	FinalSummaryPath string             `json:"finalSummaryPath,omitempty"`
	Rounds           []AutofixLoopRound `json:"rounds"`
}

type AutofixPlanProvider func(round int, previous *AutofixSummary) (*AutofixPlan, error)
type AutofixPlanExecutor func(round int, plan AutofixPlan) (string, error)

func RunAutofixLoop(policy AutofixLoopPolicy, provider AutofixPlanProvider, executor AutofixPlanExecutor) (AutofixLoopSummary, error) {
	if policy.MaxRounds <= 0 {
		policy.MaxRounds = 1
	}
	if executor == nil {
		executor = func(round int, plan AutofixPlan) (string, error) {
			return RunAutofixPlan(plan)
		}
	}
	seen := map[string]struct{}{}
	var out AutofixLoopSummary
	var previous *AutofixSummary
	for round := 1; round <= policy.MaxRounds; round++ {
		plan, err := provider(round, previous)
		if err != nil {
			return out, err
		}
		if plan == nil {
			out.Rounds = append(out.Rounds, AutofixLoopRound{Round: round, StopReason: "no_plan"})
			break
		}
		if out.Objective == "" {
			out.Objective = plan.Objective
			out.Repo = plan.Repo
		}
		fresh := []AttemptFile{}
		duplicateCount := 0
		for _, attempt := range plan.Attempts {
			sig := AttemptSignature(attempt)
			if _, ok := seen[sig]; ok {
				duplicateCount++
				continue
			}
			seen[sig] = struct{}{}
			fresh = append(fresh, attempt)
		}
		roundSummary := AutofixLoopRound{Round: round, DuplicateAttemptCount: duplicateCount}
		if len(fresh) == 0 {
			roundSummary.StopReason = "no_new_attempts"
			out.Rounds = append(out.Rounds, roundSummary)
			break
		}
		roundSummary.NewAttemptCount = len(fresh)
		for _, attempt := range fresh {
			roundSummary.AttemptNames = append(roundSummary.AttemptNames, attempt.Attempt.Name)
		}
		roundPlan := *plan
		roundPlan.Attempts = fresh
		if roundPlan.ArtifactsDir != "" {
			roundPlan.ArtifactsDir = filepath.Join(roundPlan.ArtifactsDir, fmt.Sprintf("round-%d", round))
		}
		summaryPath, err := executor(round, roundPlan)
		roundSummary.SummaryPath = summaryPath
		if summary, loadErr := LoadAutofixSummary(summaryPath); loadErr == nil {
			previous = &summary
			roundSummary.Success = summary.Success
			roundSummary.WinningAttempt = summary.WinningAttempt
			roundSummary.WinnerPromoted = summary.WinnerPromoted
			roundSummary.PromotedCheckpoint = summary.PromotedCheckpoint
			if summary.Success {
				out.Success = true
				out.WinningAttempt = summary.WinningAttempt
				out.FinalSummaryPath = summaryPath
				out.Rounds = append(out.Rounds, roundSummary)
				return out, nil
			}
		}
		if err != nil {
			roundSummary.StopReason = err.Error()
			out.Rounds = append(out.Rounds, roundSummary)
			continue
		}
		out.FinalSummaryPath = summaryPath
		out.Rounds = append(out.Rounds, roundSummary)
	}
	if !out.Success && out.FinalSummaryPath == "" && len(out.Rounds) > 0 {
		last := out.Rounds[len(out.Rounds)-1]
		if last.StopReason != "" {
			return out, fmt.Errorf("autofix loop stopped: %s", last.StopReason)
		}
	}
	if out.Success {
		return out, nil
	}
	return out, fmt.Errorf("autofix loop failed: no attempt validated")
}

func AttemptSignature(attempt AttemptFile) string {
	payload := map[string]any{
		"name":       attempt.Attempt.Name,
		"validation": attempt.Attempt.Validation,
		"mutation":   attempt.Mutation,
		"command":    attempt.Attempt.MutationCommand,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func LoadAutofixSummary(path string) (AutofixSummary, error) {
	var summary AutofixSummary
	data, err := os.ReadFile(path)
	if err != nil {
		return summary, err
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		return summary, err
	}
	return summary, nil
}
