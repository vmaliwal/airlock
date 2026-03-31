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
	avoidKinds := avoidMutationKinds(input)
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
	attempts = filterSynthesizedAttempts(attempts, avoidKinds)
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

	// Class 1: expected/got normalization mismatch
	expected, got, ok := parseExpectedGotPair(failure)
	if ok {
		if rel, line, found := findFileLineContaining(profile.TargetPath, got); found {
			newLine := strings.Replace(line, got, expected, 1)
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "normalize mismatched expected value",
				MutationKind: "replace_line",
				Confidence:   "medium",
				Rationale:    "failure text provides an expected/got pair and the target repo contains the observed value on a bounded line",
				Attempt:      AttemptFile{Attempt: AttemptSpec{Name: "normalize-mismatched-expected-value", CommitMessage: "attempt: normalize mismatched expected value", Validation: Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}}, Safety: SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 10, AllowedPaths: []string{rel}}}, Mutation: MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: rel, OldLine: line, NewLine: newLine}}},
			})
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "search replace mismatched expected value",
				MutationKind: "search_replace",
				Confidence:   "low",
				Rationale:    "fallback variant of the same bounded repair using search_replace so later rounds can switch mutation families",
				Attempt:      AttemptFile{Attempt: AttemptSpec{Name: "search-replace-mismatched-expected-value", CommitMessage: "attempt: search replace mismatched expected value", Validation: Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}}, Safety: SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 10, AllowedPaths: []string{rel}}}, Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: rel, OldText: line, NewText: newLine}}},
			})
		}
	}

	// Class 2: resource lifecycle — missing defer close (os.Create / os.Open not closed)
	if resourceLifecycleSignal(failure) {
		if rel, ctx, found := findFileMultiLineContext(profile.TargetPath, []string{"os.Create(", "return "}, 5); found {
			anchorLine := resourceLifecycleAnchor(ctx, "os.Create(")
			if anchorLine != "" {
				attempts = append(attempts, SynthesizedAttempt{
					Name:         "defer close resource after create",
					MutationKind: "insert_after",
					Confidence:   "medium",
					Rationale:    "failure text describes a file-descriptor or resource leak; inserting defer w.Close() after os.Create is a bounded repair for the common missing-close pattern",
					Attempt: AttemptFile{
						Attempt: AttemptSpec{
							Name:          "defer-close-resource-after-create",
							CommitMessage: "attempt: defer close resource after create",
							Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
							Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 5, AllowedPaths: []string{rel}},
						},
						Mutation: MutationSpec{InsertAfter: &InsertAfterMutation{
							Path:       rel,
							AnchorText: anchorLine,
							InsertText: "\n\tdefer w.Close()",
						}},
					},
				})
			}
		}
		// Also try os.Open variant
		if rel, ctx, found := findFileMultiLineContext(profile.TargetPath, []string{"os.Open(", "return "}, 5); found {
			anchorLine := resourceLifecycleAnchor(ctx, "os.Open(")
			if anchorLine != "" {
				attempts = append(attempts, SynthesizedAttempt{
					Name:         "defer close file after open",
					MutationKind: "insert_after",
					Confidence:   "medium",
					Rationale:    "failure text describes a resource leak; inserting defer f.Close() after os.Open is a bounded repair",
					Attempt: AttemptFile{
						Attempt: AttemptSpec{
							Name:          "defer-close-file-after-open",
							CommitMessage: "attempt: defer close file after open",
							Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
							Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 5, AllowedPaths: []string{rel}},
						},
						Mutation: MutationSpec{InsertAfter: &InsertAfterMutation{
							Path:       rel,
							AnchorText: anchorLine,
							InsertText: "\n\tdefer f.Close()",
						}},
					},
				})
			}
		}
	}

	return attempts
}

// resourceLifecycleSignal returns true when the failure text describes a
// resource lifecycle bug: fd leak, missing close, defer close, etc.
func resourceLifecycleSignal(text string) bool {
	lower := strings.ToLower(text)
	signals := []string{
		"file descriptor leak", "fd leak", "resource leak",
		"never closed", "not closed", "missing close", "missing defer",
		"defer close", "defer w.close", "defer f.close",
		"os.create", "os.open", "writesnapshot", "writemanifest",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// resourceLifecycleAnchor finds the full source line containing the given call
// inside a context window string. Returns the trimmed line.
func resourceLifecycleAnchor(ctx, call string) string {
	for _, line := range strings.Split(ctx, "\n") {
		if strings.Contains(line, call) {
			trimmed := strings.TrimRight(line, " \t")
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func synthesizePythonAttempts(input PlanInput, profile RepoProfile, validationCmd string) []SynthesizedAttempt {
	failure := strings.ToLower(input.FailureText + "\n" + input.Notes)
	attempts := []SynthesizedAttempt{}

	// Class 1: unclosed code-block EOF preservation (LangChain-style)
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

	// Class 2: empty-string reasoning content guard (LangChain-style)
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

	// Class 3: generic isinstance/None type-guard narrowing
	// Matches bugs where a None-check is missing before an isinstance check or
	// vice versa, causing AttributeError / TypeError on None or wrong type.
	if pythonTypeGuardSignal(failure) {
		attempts = append(attempts, synthesizePythonTypeGuardAttempts(profile, validationCmd)...)
	}

	// Class 4: return-value preservation at function exit — missing return of
	// accumulated buffer/list/dict at the end of a processing function.
	if strings.Contains(failure, "missing") || strings.Contains(failure, "discards") || strings.Contains(failure, "drops") || strings.Contains(failure, "lost") {
		attempts = append(attempts, synthesizePythonMissingReturnAttempts(profile, validationCmd)...)
	}

	return attempts
}

// pythonTypeGuardSignal detects failure text that suggests a type or None guard issue.
func pythonTypeGuardSignal(lower string) bool {
	signals := []string{
		"nonetype", "attributeerror", "typeerror",
		"none check", "isinstance", "type guard",
		"should be none", "should not be none",
		"is none when", "is not none",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// synthesizePythonTypeGuardAttempts generates bounded type-guard repair attempts.
func synthesizePythonTypeGuardAttempts(profile RepoProfile, validationCmd string) []SynthesizedAttempt {
	attempts := []SynthesizedAttempt{}
	// Look for patterns like: `if x is not None and isinstance(x, SomeType):`
	// where the fix is typically to add `and x` or reorder checks.
	needles := []string{"is not None", "isinstance("}
	if rel, ok := findFileContaining(profile.TargetPath, needles); ok {
		attempts = append(attempts, SynthesizedAttempt{
			Name:         "tighten none and type guard",
			MutationKind: "search_replace",
			Confidence:   "low",
			Rationale:    "failure text suggests a type or None guard mismatch; a search_replace to add a truthiness check alongside isinstance is a bounded candidate",
			Attempt: AttemptFile{
				Attempt: AttemptSpec{
					Name:          "tighten-none-and-type-guard",
					CommitMessage: "attempt: tighten none and type guard",
					Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
					Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 5, AllowedPaths: []string{rel}},
				},
				// Placeholder: planner will provide exact old/new text when
				// configured; the attempt file will fail safely if text not found.
				Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{
					Path:    rel,
					OldText: "if x is not None and isinstance(x,",
					NewText: "if isinstance(x,",
				}},
			},
		})
	}
	return attempts
}

// synthesizePythonMissingReturnAttempts handles the class of bugs where a
// function silently returns None instead of returning accumulated content.
func synthesizePythonMissingReturnAttempts(profile RepoProfile, validationCmd string) []SynthesizedAttempt {
	attempts := []SynthesizedAttempt{}
	// Look for patterns: function body with an accumulator variable that is
	// not returned at the end (returns None implicitly).
	accumulatorNeedles := [][]string{
		{"result = []", "return"},
		{"result = {}", "return"},
		{"chunks = []", "return"},
		{"output = []", "return"},
		{"items = []", "return"},
	}
	for _, needles := range accumulatorNeedles {
		if rel, ok := findFileContaining(profile.TargetPath, needles); ok {
			varName := strings.TrimSuffix(strings.TrimSuffix(strings.Split(needles[0], " = ")[0], " "), "\t")
			attempts = append(attempts, SynthesizedAttempt{
				Name:         "return accumulated " + varName + " at function exit",
				MutationKind: "replace_line",
				Confidence:   "low",
				Rationale:    "failure describes lost/dropped content; function may be missing return of its accumulator variable",
				Attempt: AttemptFile{
					Attempt: AttemptSpec{
						Name:          "return-accumulated-" + strings.ToLower(varName),
						CommitMessage: "attempt: return accumulated " + varName + " at function exit",
						Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
						Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 5, AllowedPaths: []string{rel}},
					},
					Mutation: MutationSpec{ReplaceLine: &ReplaceLineMutation{
						Path:    rel,
						OldLine: "        return",
						NewLine: "        return " + varName,
					}},
				},
			})
			break
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
