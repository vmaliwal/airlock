package research

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmaliwal/airlock/internal/util"
)

type AttemptSpec struct {
	Name            string        `json:"name"`
	MutationCommand string        `json:"mutationCommand"`
	Validation      Phase         `json:"validation"`
	Safety          SafetyBudget  `json:"safety,omitempty"`
	CommitMessage   string        `json:"commit_message,omitempty"`
	ResetOnSuccess  bool          `json:"reset_on_success,omitempty"`
	Timeout         time.Duration `json:"-"`
}

type AttemptOutcome struct {
	Name                   string               `json:"name"`
	BaseSHA                string               `json:"baseSha"`
	Mutation               CommandResult        `json:"mutation"`
	DiffStats              GitDiffStats         `json:"diffStats"`
	BudgetErrors           []string             `json:"budgetErrors,omitempty"`
	ValidationRuns         []CommandResult      `json:"validationRuns,omitempty"`
	ValidationEval         EvaluationResult     `json:"validationEval,omitempty"`
	ValidationFingerprints []FailureFingerprint `json:"validationFingerprints,omitempty"`
	PatchPath              string               `json:"patchPath,omitempty"`
	Success                bool                 `json:"success"`
	ResetApplied           bool                 `json:"resetApplied"`
}

func RunLocalCommand(repo, cmd string, timeout time.Duration) (CommandResult, error) {
	start := time.Now()
	out, err := util.RunLocal("bash", []string{"-lc", cmd}, util.RunOptions{Cwd: repo, Timeout: timeout})
	duration := time.Since(start).Milliseconds()
	result := CommandResult{Command: cmd, DurationMs: duration}
	if err != nil {
		msg := err.Error()
		if idx := strings.LastIndex(msg, "\n"); idx >= 0 {
			result.Stderr = msg[:idx]
			result.Stdout = msg[idx+1:]
		} else {
			result.Stderr = msg
		}
		result.ExitCode = 1
		return result, nil
	}
	result.Stdout = string(out)
	result.ExitCode = 0
	return result, nil
}

func EnforceSafetyBudget(diff GitDiffStats, safety SafetyBudget) []string {
	errs := []string{}
	if safety.MaxFilesChanged > 0 && diff.FilesChangedCount > safety.MaxFilesChanged {
		errs = append(errs, fmt.Sprintf("files_changed %d > %d", diff.FilesChangedCount, safety.MaxFilesChanged))
	}
	if safety.MaxLocChanged > 0 && diff.LocChanged > safety.MaxLocChanged {
		errs = append(errs, fmt.Sprintf("loc_changed %d > %d", diff.LocChanged, safety.MaxLocChanged))
	}
	if len(safety.AllowedPaths) > 0 {
		outside := []string{}
		for _, p := range diff.ChangedFiles {
			if !matchesAny(p, safety.AllowedPaths) {
				outside = append(outside, p)
			}
		}
		if len(outside) > 0 {
			errs = append(errs, "paths outside allowlist: "+strings.Join(outside, ", "))
		}
	}
	if len(safety.ForbiddenPaths) > 0 {
		bad := []string{}
		for _, p := range diff.ChangedFiles {
			if matchesAny(p, safety.ForbiddenPaths) {
				bad = append(bad, p)
			}
		}
		if len(bad) > 0 {
			errs = append(errs, "paths in forbidden list: "+strings.Join(bad, ", "))
		}
	}
	return errs
}

func RunNativeAttempt(repo, artifactsDir, checkpointSHA string, spec AttemptSpec) (AttemptOutcome, error) {
	return RunNativeAttemptWithMutation(repo, artifactsDir, checkpointSHA, spec, MutationSpec{})
}

func RunNativeAttemptWithMutation(repo, artifactsDir, checkpointSHA string, spec AttemptSpec, mutationSpec MutationSpec) (AttemptOutcome, error) {
	excludes := artifactExcludes(repo, artifactsDir)
	if err := GitResetHardTo(repo, checkpointSHA); err != nil {
		return AttemptOutcome{}, err
	}
	if err := GitCleanExcept(repo, excludes); err != nil {
		return AttemptOutcome{}, err
	}
	outcome := AttemptOutcome{Name: spec.Name, BaseSHA: checkpointSHA, ResetApplied: true}
	var mutation CommandResult
	var err error
	if mutationSpec.SearchReplace != nil || mutationSpec.InsertAfter != nil || mutationSpec.ReplaceLine != nil || mutationSpec.CreateFile != nil || mutationSpec.ApplyPatch != nil || mutationSpec.EnsureLine != nil || mutationSpec.NilGuard != nil || mutationSpec.ErrorReturn != nil {
		mutation, err = ApplyMutationSpec(repo, mutationSpec)
	} else {
		mutation, err = RunLocalCommand(repo, spec.MutationCommand, spec.Timeout)
	}
	if err != nil {
		return outcome, err
	}
	outcome.Mutation = mutation
	if mutation.ExitCode != 0 {
		_ = GitResetAttemptExcept(repo, excludes)
		return outcome, nil
	}
	diff, err := GitDiffNumstat(repo)
	if err != nil {
		_ = GitResetAttemptExcept(repo, excludes)
		return outcome, err
	}
	outcome.DiffStats = diff
	if diff.FilesChangedCount == 0 {
		outcome.BudgetErrors = append(outcome.BudgetErrors, "patch produced no diff")
	}
	outcome.BudgetErrors = append(outcome.BudgetErrors, EnforceSafetyBudget(diff, spec.Safety)...)
	patchPath := filepath.Join(artifactsDir, spec.Name+".patch")
	if err := GitWritePatch(repo, patchPath); err == nil {
		outcome.PatchPath = patchPath
	}
	if len(outcome.BudgetErrors) > 0 {
		_ = GitResetAttemptExcept(repo, excludes)
		return outcome, nil
	}
	repeat := spec.Validation.Repeat
	if repeat <= 0 {
		repeat = 1
	}
	for i := 0; i < repeat; i++ {
		res, err := RunLocalCommand(repo, spec.Validation.Command, spec.Timeout)
		if err != nil {
			_ = GitResetAttemptExcept(repo, excludes)
			return outcome, err
		}
		outcome.ValidationRuns = append(outcome.ValidationRuns, res)
	}
	outcome.ValidationEval = EvaluateRepeated(outcome.ValidationRuns, spec.Validation.Success)
	outcome.ValidationFingerprints = SummarizeFailures(outcome.ValidationRuns)
	outcome.Success = outcome.ValidationEval.Passed
	if !outcome.Success {
		_ = GitResetAttemptExcept(repo, excludes)
		return outcome, nil
	}
	if spec.CommitMessage != "" {
		if err := GitCommitAll(repo, spec.CommitMessage); err != nil {
			return outcome, err
		}
	}
	if spec.ResetOnSuccess {
		_ = GitResetAttemptExcept(repo, excludes)
	}
	return outcome, nil
}

func artifactExcludes(repo, artifactsDir string) []string {
	rel, err := filepath.Rel(repo, artifactsDir)
	if err != nil {
		return nil
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return nil
	}
	return []string{rel}
}

func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
	}
	return false
}
