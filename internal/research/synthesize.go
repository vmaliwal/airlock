package research

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmaliwal/airlock/internal/util"
)

type SynthesizedAttempt struct {
	Name         string      `json:"name"`
	MutationKind string      `json:"mutationKind"`
	Confidence   string      `json:"confidence"`
	Rationale    string      `json:"rationale"`
	Attempt      AttemptFile `json:"attempt"`
}

type SynthesisReport struct {
	Input         PlanInput            `json:"input"`
	Investigation InvestigationReport  `json:"investigation"`
	Supported     bool                 `json:"supported"`
	Summary       string               `json:"summary"`
	Attempts      []SynthesizedAttempt `json:"attempts,omitempty"`
	AutofixPlan   *AutofixPlan         `json:"autofixPlan,omitempty"`
}

func SynthesizeAutofixPlan(input PlanInput, vmBackend string, allowHostExecution bool) (SynthesisReport, error) {
	report, err := PlanFromInput(input, vmBackend, allowHostExecution)
	if err != nil {
		return SynthesisReport{}, err
	}
	profile := report.Investigation.Profile
	validationCmd := applyRuntimeBootstrapPolicy(profile, compiledTargetCommand(input, report))
	attempts := []SynthesizedAttempt{}
	summary := ""
	if planner, enabled, err := plannerFactory(); err != nil {
		return SynthesisReport{}, err
	} else if enabled {
		attempts, summary, err = synthesizeWithPlanner(context.Background(), planner, input, report, validationCmd)
		if err != nil {
			return SynthesisReport{}, err
		}
	} else {
		attempts, summary = synthesizeAttemptsForInput(input, profile, validationCmd)
	}
	resp := SynthesisReport{
		Input:         input,
		Investigation: report.Investigation,
		Supported:     len(attempts) > 0,
		Summary:       summary,
		Attempts:      attempts,
	}
	if len(attempts) == 0 {
		return resp, nil
	}
	artifactsDir := filepath.ToSlash(filepath.Join("/tmp", "airlock-synth-"+util.SafeName(compiledObjective(input))))
	autofix := AutofixPlan{
		Objective:        compiledObjective(input),
		Repo:             profile.TargetPath,
		ArtifactsDir:     artifactsDir,
		FingerprintHints: collectFingerprintHintsFromFailureText(input.FailureText),
		Attempts:         []AttemptFile{},
	}
	for _, item := range attempts {
		autofix.Attempts = append(autofix.Attempts, item.Attempt)
	}
	resp.AutofixPlan = &autofix
	return resp, nil
}

func synthesizeAttemptsForInput(input PlanInput, profile RepoProfile, validationCmd string) ([]SynthesizedAttempt, string) {
	if profile.RepoType == "go" {
		if attempts := synthesizeGoAttempts(input, profile, validationCmd); len(attempts) > 0 {
			return attempts, "generated candidate attempts for a supported Go bug-class heuristic"
		}
	}
	if profile.RepoType == "python" {
		if attempts := synthesizePythonAttempts(input, profile, validationCmd); len(attempts) > 0 {
			return attempts, "generated candidate attempts for a supported Python bug-class heuristic"
		}
	}
	return nil, "no supported structured synthesis heuristic matched this bug signal yet"
}

func synthesizeGoAttempts(input PlanInput, profile RepoProfile, validationCmd string) []SynthesizedAttempt {
	failure := input.FailureText + "\n" + input.Notes
	attempts := []SynthesizedAttempt{}
	expected, got, ok := parseExpectedGotPair(failure)
	if ok {
		if rel, line, found := findFileLineContaining(profile.TargetPath, got); found {
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "normalize mismatched expected value",
				MutationKind: "replace_line",
				Confidence:   "medium",
				Rationale:    "failure text provides an expected/got pair and the target repo contains the observed value on a bounded line",
				Attempt:      AttemptFile{Attempt: AttemptSpec{Name: "normalize-mismatched-expected-value", CommitMessage: "attempt: normalize mismatched expected value", Validation: Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}}, Safety: SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 10, AllowedPaths: []string{rel}}}, Mutation: MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: rel, OldLine: line, NewLine: strings.Replace(line, got, expected, 1)}}},
			})
		}
	}
	return attempts
}

func synthesizePythonAttempts(input PlanInput, profile RepoProfile, validationCmd string) []SynthesizedAttempt {
	failure := strings.ToLower(input.FailureText + "\n" + input.Notes)
	attempts := []SynthesizedAttempt{}
	if strings.Contains(failure, "unclosed code block") || strings.Contains(failure, "_resolve_code_chunk") {
		if rel, ok := findFileContaining(profile.TargetPath, []string{"_resolve_code_chunk", "return \"\""}); ok {
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "preserve accumulated code chunk at eof",
				MutationKind: "replace_line",
				Confidence:   "high",
				Rationale:    "failure text points to EOF dropping accumulated code-block content; replacing the empty-string return is a bounded repair",
				Attempt: AttemptFile{
					Attempt: AttemptSpec{
						Name:          "preserve-accumulated-code-chunk-at-eof",
						CommitMessage: "attempt: preserve accumulated code chunk at eof",
						Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
						Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 10, AllowedPaths: []string{rel}},
					},
					Mutation: MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: rel, OldLine: "        return \"\"", NewLine: "        return chunk"}},
				},
			})
		}
	}
	if strings.Contains(failure, "empty string") || strings.Contains(failure, "reasoning_content") {
		if rel, ok := findFileContaining(profile.TargetPath, []string{"reasoning_content", "is not None", "isinstance(reasoning_content, str)"}); ok {
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "ignore empty reasoning_content strings",
				MutationKind: "search_replace",
				Confidence:   "medium",
				Rationale:    "failure text indicates empty strings should be ignored, so tightening the string guard is a bounded candidate repair",
				Attempt: AttemptFile{
					Attempt: AttemptSpec{
						Name:          "ignore-empty-reasoning-content-strings",
						CommitMessage: "attempt: ignore empty reasoning content strings",
						Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
						Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 20, AllowedPaths: []string{rel}},
					},
					Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{
						Path:    rel,
						OldText: "if reasoning_content is not None and isinstance(reasoning_content, str):",
						NewText: "if isinstance(reasoning_content, str) and reasoning_content:",
					}},
				},
			})
		}
	}
	return attempts
}

func findFileContaining(root string, needles []string) (string, bool) {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".py") && !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".js") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				return nil
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		found = filepath.ToSlash(rel)
		return fmt.Errorf("found")
	})
	return found, found != ""
}
