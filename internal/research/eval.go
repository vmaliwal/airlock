package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PlannerEvalCase struct {
	Name                  string    `json:"name"`
	Input                 PlanInput `json:"input"`
	ExpectedSupported     *bool     `json:"expectedSupported,omitempty"`
	ExpectedMutationKinds []string  `json:"expectedMutationKinds,omitempty"`
	ExecuteAutofix        bool      `json:"executeAutofix,omitempty"`
	ExpectAutofixSuccess  *bool     `json:"expectAutofixSuccess,omitempty"`
}

type PlannerEvalCaseResult struct {
	Name             string   `json:"name"`
	Supported        bool     `json:"supported"`
	SchemaValid      bool     `json:"schemaValid"`
	AttemptCount     int      `json:"attemptCount"`
	Top1MutationKind string   `json:"top1MutationKind,omitempty"`
	Top3Kinds        []string `json:"top3Kinds,omitempty"`
	ExpectedKinds    []string `json:"expectedKinds,omitempty"`
	Top1Hit          bool     `json:"top1Hit,omitempty"`
	Top3Hit          bool     `json:"top3Hit,omitempty"`
	AutofixExecuted  bool     `json:"autofixExecuted,omitempty"`
	AutofixSuccess   bool     `json:"autofixSuccess,omitempty"`
	AutofixSummary   string   `json:"autofixSummary,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

type PlannerEvalSummary struct {
	StartedAt            string                  `json:"startedAt"`
	FinishedAt           string                  `json:"finishedAt"`
	CaseCount            int                     `json:"caseCount"`
	SupportedCount       int                     `json:"supportedCount"`
	SchemaValidCount     int                     `json:"schemaValidCount"`
	Top1MeasuredCount    int                     `json:"top1MeasuredCount"`
	Top1HitCount         int                     `json:"top1HitCount"`
	Top3MeasuredCount    int                     `json:"top3MeasuredCount"`
	Top3HitCount         int                     `json:"top3HitCount"`
	AutofixMeasuredCount int                     `json:"autofixMeasuredCount"`
	AutofixSuccessCount  int                     `json:"autofixSuccessCount"`
	Results              []PlannerEvalCaseResult `json:"results"`
}

func LoadPlannerEvalCases(path string) ([]PlannerEvalCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cases []PlannerEvalCase
	if err := json.Unmarshal(data, &cases); err == nil {
		return cases, nil
	}
	var wrapper struct {
		Cases []PlannerEvalCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Cases, nil
}

func RunPlannerEvalCases(cases []PlannerEvalCase, vmBackend string, allowHostExecution bool) (PlannerEvalSummary, error) {
	started := time.Now().UTC()
	summary := PlannerEvalSummary{StartedAt: started.Format(time.RFC3339), Results: []PlannerEvalCaseResult{}}
	for _, c := range cases {
		res := PlannerEvalCaseResult{Name: c.Name, ExpectedKinds: append([]string{}, c.ExpectedMutationKinds...)}
		report, err := SynthesizeAutofixPlan(c.Input, vmBackend, allowHostExecution)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			summary.Results = append(summary.Results, res)
			continue
		}
		res.Supported = report.Supported
		if report.Supported {
			summary.SupportedCount++
		}
		res.AttemptCount = len(report.Attempts)
		res.SchemaValid = plannerAttemptsSchemaValid(report)
		if res.SchemaValid {
			summary.SchemaValidCount++
		}
		for i, item := range report.Attempts {
			if i == 0 {
				res.Top1MutationKind = item.MutationKind
			}
			if i < 3 {
				res.Top3Kinds = append(res.Top3Kinds, item.MutationKind)
			}
		}
		if len(c.ExpectedMutationKinds) > 0 {
			summary.Top1MeasuredCount++
			res.Top1Hit = contains(c.ExpectedMutationKinds, res.Top1MutationKind)
			if res.Top1Hit {
				summary.Top1HitCount++
			}
			summary.Top3MeasuredCount++
			for _, kind := range res.Top3Kinds {
				if contains(c.ExpectedMutationKinds, kind) {
					res.Top3Hit = true
					break
				}
			}
			if res.Top3Hit {
				summary.Top3HitCount++
			}
		}
		if c.ExpectedSupported != nil && *c.ExpectedSupported != report.Supported {
			res.Errors = append(res.Errors, fmt.Sprintf("expectedSupported=%v got %v", *c.ExpectedSupported, report.Supported))
		}
		if c.ExecuteAutofix && report.AutofixPlan != nil {
			summary.AutofixMeasuredCount++
			res.AutofixExecuted = true
			plan := *report.AutofixPlan
			copyRoot, err := cloneLocalRepoForEval(report.Investigation.Profile.TargetPath)
			if err != nil {
				res.Errors = append(res.Errors, err.Error())
			} else {
				plan.Repo = copyRoot
				plan.ArtifactsDir = filepath.Join(copyRoot, ".airlock-eval-artifacts")
				summaryPath, runErr := RunAutofixPlan(plan)
				res.AutofixSummary = summaryPath
				res.AutofixSuccess = runErr == nil
				if res.AutofixSuccess {
					summary.AutofixSuccessCount++
				} else if runErr != nil {
					res.Errors = append(res.Errors, runErr.Error())
				}
				if c.ExpectAutofixSuccess != nil && *c.ExpectAutofixSuccess != res.AutofixSuccess {
					res.Errors = append(res.Errors, fmt.Sprintf("expectedAutofixSuccess=%v got %v", *c.ExpectAutofixSuccess, res.AutofixSuccess))
				}
			}
		}
		summary.Results = append(summary.Results, res)
	}
	summary.CaseCount = len(cases)
	summary.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	return summary, nil
}

func plannerAttemptsSchemaValid(report SynthesisReport) bool {
	if !report.Supported {
		return true
	}
	for _, item := range report.Attempts {
		errs := ValidateAttemptFile(AttemptFile{Repo: report.Investigation.Profile.TargetPath, ArtifactsDir: "/tmp/airlock-eval", Attempt: item.Attempt.Attempt, Mutation: item.Attempt.Mutation})
		if len(errs) > 0 {
			return false
		}
	}
	return true
}

func cloneLocalRepoForEval(path string) (string, error) {
	gitRoot, err := GitTopLevel(path)
	if err != nil {
		return "", err
	}
	copyRoot, err := os.MkdirTemp("", "airlock-eval-")
	if err != nil {
		return "", err
	}
	if _, err := RunLocalCommand(filepath.Dir(copyRoot), fmt.Sprintf("git clone %s %s", shellEscape(gitRoot), shellEscape(copyRoot)), 5*time.Minute); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(gitRoot, path)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return copyRoot, nil
	}
	return filepath.Join(copyRoot, rel), nil
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
