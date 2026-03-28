package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type runArtifacts struct {
	ArtifactsDir string
	RepoDir      string
}

func ExecuteRunContract(c RunContract, repoRoot, artifactsDir string) error {
	repo := repoRoot
	if c.TargetPath != "" {
		repo = filepath.Join(repoRoot, c.TargetPath)
	}
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return err
	}
	_ = writeJSON(filepath.Join(artifactsDir, "execution-policy.json"), map[string]any{
		"hostExecutionException": c.HostExecutionException,
		"planPresent":            c.Plan != nil,
		"targetPath":             c.TargetPath,
		"repoRoot":               repoRoot,
		"effectiveRepo":          repo,
	})
	if err := GitEnsureIdentity(repo); err != nil {
		return err
	}
	if c.Baseline != nil {
		baseline, err := runPhase(repo, c.Baseline.Command, 600*time.Second)
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(artifactsDir, "baseline-results.json"), baseline); err != nil {
			return err
		}
	}
	attemptLog := filepath.Join(artifactsDir, "attempt-log.jsonl")
	for _, step := range c.Setup {
		result, err := runPhase(repo, step.Command, 600*time.Second)
		if err != nil {
			return err
		}
		_ = appendJSONL(attemptLog, map[string]any{"stage": "setup", "step": step.Name, "result": result})
		if result.ExitCode != 0 {
			_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: setup step failed\n")
			return fmt.Errorf("setup step failed: %s", step.Name)
		}
		if step.CommitMessage != "" {
			if err := GitCommitAll(repo, step.CommitMessage); err != nil {
				return err
			}
		}
	}

	reproRuns, reproEval, reproFPS, err := runRepeatedPhase(repo, c.Reproduction, 600*time.Second)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(artifactsDir, "reproduction-results.json"), map[string]any{"runs": reproRuns, "evaluation": reproEval, "fingerprints": reproFPS}); err != nil {
		return err
	}
	if !reproEval.Passed {
		_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: reproduction criteria not met\n")
		return fmt.Errorf("reproduction criteria not met")
	}
	if c.Mode == "read_only" {
		return writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nRead-only run complete.\n")
	}

	bestFailureCount := int(^uint(0) >> 1)
	attemptsWithoutImprovement := 0
	repeatedTop := 0
	previousTop := ""
	var baselineCampaign map[string]any
	if c.Campaign != nil {
		inv, err := runPhase(repo, c.Campaign.InventoryCommand, 20*time.Minute)
		if err != nil {
			return err
		}
		summary := SummarizeFailures([]CommandResult{inv})
		baselineCampaign = map[string]any{"result": inv, "summary": summary, "totalFailures": countFingerprints(summary)}
		bestFailureCount = countFingerprints(summary)
		if err := writeJSON(filepath.Join(artifactsDir, "campaign-baseline.json"), baselineCampaign); err != nil {
			return err
		}
	}

	success := false
	for i, patch := range c.Patches {
		patchResult, err := runPhase(repo, patch.Command, 10*time.Minute)
		if err != nil {
			return err
		}
		if patchResult.ExitCode != 0 {
			_ = appendJSONL(attemptLog, map[string]any{"attempt": i + 1, "patch": patch.Name, "patchResult": patchResult, "outcome": "patch_command_failed"})
			_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: patch command failed\n")
			return fmt.Errorf("patch command failed: %s", patch.Name)
		}
		diff, err := GitDiffNumstat(repo)
		if err != nil {
			return err
		}
		budgetErrors := EnforceSafetyBudget(diff, c.Safety)
		if diff.FilesChangedCount == 0 {
			budgetErrors = append(budgetErrors, "patch produced no diff")
		}
		_ = GitWritePatch(repo, filepath.Join(artifactsDir, "repo.patch"))
		if len(budgetErrors) > 0 {
			_ = appendJSONL(attemptLog, map[string]any{"attempt": i + 1, "patch": patch.Name, "patchResult": patchResult, "diffStats": diff, "budgetErrors": budgetErrors})
			_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: patch rejected\n")
			return fmt.Errorf("patch rejected")
		}
		valRuns, valEval, valFPS, err := runRepeatedValidation(repo, c.Validation.TargetCommand, c.Validation.Repeat, c.Validation.Success, 20*time.Minute)
		if err != nil {
			return err
		}
		neighborEval := EvaluationResult{Passed: true}
		var neighborRun *CommandResult
		if c.Validation.NeighborCommand != "" {
			nr, err := runPhase(repo, c.Validation.NeighborCommand, 20*time.Minute)
			if err != nil {
				return err
			}
			neighborRun = &nr
			rule := c.Validation.NeighborSuccess
			neighborEval = EvaluateSingle(nr, rule)
		}
		broaderRuns := []map[string]any{}
		broaderPassed := true
		for _, item := range c.Validation.BroaderCommands {
			br, err := runPhase(repo, item.Command, 20*time.Minute)
			if err != nil {
				return err
			}
			be := EvaluateSingle(br, item.Success)
			broaderRuns = append(broaderRuns, map[string]any{"name": item.Name, "run": br, "eval": be})
			broaderPassed = broaderPassed && be.Passed
		}
		campaignEval := EvaluationResult{Passed: true}
		var campaignResult map[string]any
		currentFailures := valEval.FailCount
		top := "success"
		if len(valFPS) > 0 {
			top = valFPS[0].Signature
		}
		if c.Campaign != nil {
			cr, err := runPhase(repo, c.Campaign.InventoryCommand, 20*time.Minute)
			if err != nil {
				return err
			}
			summary := SummarizeFailures([]CommandResult{cr})
			campaignResult = map[string]any{"result": cr, "summary": summary, "totalFailures": countFingerprints(summary)}
			campaignEval = evaluateCampaign(summary, baselineCampaign, c.Campaign.Success)
			currentFailures = countFingerprints(summary)
			if len(summary) > 0 {
				top = summary[0].Signature
			} else {
				top = "success"
			}
		}
		if currentFailures < bestFailureCount {
			bestFailureCount = currentFailures
			attemptsWithoutImprovement = 0
		} else {
			attemptsWithoutImprovement++
		}
		if top == previousTop {
			repeatedTop++
		} else {
			repeatedTop = 1
		}
		previousTop = top
		_ = appendJSONL(attemptLog, map[string]any{"attempt": i + 1, "patch": patch.Name, "patchResult": patchResult, "diffStats": diff, "validationEval": valEval, "validationFingerprints": valFPS, "neighborEval": neighborEval, "broaderRuns": broaderRuns, "campaignResult": campaignResult, "campaignEval": campaignEval, "attemptsWithoutImprovement": attemptsWithoutImprovement, "repeatedTopFingerprint": repeatedTop})
		if c.Stuck.MaxSameFailureFingerprint > 0 && repeatedTop >= c.Stuck.MaxSameFailureFingerprint {
			_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: stuck on same failure fingerprint\n")
			return fmt.Errorf("stuck on same fingerprint")
		}
		if c.Stuck.MaxAttemptsWithoutImprovement > 0 && attemptsWithoutImprovement >= c.Stuck.MaxAttemptsWithoutImprovement {
			_ = writeText(filepath.Join(artifactsDir, "outcome.md"), "# Outcome\n\nSuccess: false\n\nReason: no improvement within attempt budget\n")
			return fmt.Errorf("no improvement")
		}
		if valEval.Passed && neighborEval.Passed && broaderPassed && campaignEval.Passed {
			payload := map[string]any{"attempt": i + 1, "patch": patch.Name, "diffStats": diff, "validationRuns": valRuns, "validationEval": valEval, "validationFingerprints": valFPS, "broaderRuns": broaderRuns, "campaignResult": campaignResult, "campaignEval": campaignEval}
			if neighborRun != nil {
				payload["neighborRun"] = neighborRun
				payload["neighborEval"] = neighborEval
			}
			if err := writeJSON(filepath.Join(artifactsDir, "validation-results.json"), payload); err != nil {
				return err
			}
			success = true
			break
		}
	}
	return writeText(filepath.Join(artifactsDir, "outcome.md"), fmt.Sprintf("# Outcome\n\nSuccess: %v\n", success))
}

func runPhase(repo, cmd string, timeout time.Duration) (CommandResult, error) {
	return RunLocalCommand(repo, cmd, timeout)
}

func runRepeatedPhase(repo string, phase Phase, timeout time.Duration) ([]CommandResult, EvaluationResult, []FailureFingerprint, error) {
	return runRepeatedValidation(repo, phase.Command, phase.Repeat, phase.Success, timeout)
}

func runRepeatedValidation(repo, cmd string, repeat int, rule SuccessRule, timeout time.Duration) ([]CommandResult, EvaluationResult, []FailureFingerprint, error) {
	if repeat <= 0 {
		repeat = 1
	}
	runs := make([]CommandResult, 0, repeat)
	for i := 0; i < repeat; i++ {
		res, err := runPhase(repo, cmd, timeout)
		if err != nil {
			return nil, EvaluationResult{}, nil, err
		}
		runs = append(runs, res)
	}
	eval := EvaluateRepeated(runs, rule)
	fps := SummarizeFailures(runs)
	return runs, eval, fps, nil
}

func countFingerprints(items []FailureFingerprint) int {
	total := 0
	for _, item := range items {
		total += item.Count
	}
	return total
}

func evaluateCampaign(current []FailureFingerprint, baseline map[string]any, cfg CampaignSuccess) EvaluationResult {
	observed := []string{fmt.Sprintf("total_failures=%d", countFingerprints(current))}
	failed := []string{}
	if cfg.MaxTotalFailures != nil && countFingerprints(current) > *cfg.MaxTotalFailures {
		failed = append(failed, fmt.Sprintf("total_failures > %d", *cfg.MaxTotalFailures))
	}
	currentSet := map[string]struct{}{}
	for _, fp := range current {
		currentSet[fp.Signature] = struct{}{}
	}
	for _, req := range cfg.MustRemoveFingerprints {
		if _, ok := currentSet[req]; ok {
			failed = append(failed, "fingerprint still present: "+req)
		}
	}
	if cfg.MustNotIntroduceFingerprints && baseline != nil {
		baseSet := map[string]struct{}{}
		if baseItems, ok := baseline["summary"].([]FailureFingerprint); ok {
			for _, fp := range baseItems {
				baseSet[fp.Signature] = struct{}{}
			}
		}
		introduced := []string{}
		for sig := range currentSet {
			if _, ok := baseSet[sig]; !ok {
				introduced = append(introduced, sig)
			}
		}
		if len(introduced) > 0 {
			failed = append(failed, "introduced fingerprints: "+stringsJoin(introduced))
		}
	}
	return EvaluationResult{Passed: len(failed) == 0, Observed: observed, FailedRules: failed}
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
func writeText(path, s string) error { return os.WriteFile(path, []byte(s), 0o644) }
func appendJSONL(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}
