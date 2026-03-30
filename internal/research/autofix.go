package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type AutofixPlan struct {
	Objective        string        `json:"objective"`
	Repo             string        `json:"repo"`
	ArtifactsDir     string        `json:"artifactsDir"`
	Checkpoint       string        `json:"checkpoint,omitempty"`
	FingerprintHints []string      `json:"fingerprint_hints,omitempty"`
	Attempts         []AttemptFile `json:"attempts"`
}

type AutofixSummary struct {
	Objective          string           `json:"objective"`
	Repo               string           `json:"repo"`
	StartedAt          string           `json:"startedAt"`
	FinishedAt         string           `json:"finishedAt"`
	Success            bool             `json:"success"`
	WinningAttempt     string           `json:"winningAttempt,omitempty"`
	WinnerPromoted     bool             `json:"winnerPromoted,omitempty"`
	PromotedCheckpoint string           `json:"promotedCheckpoint,omitempty"`
	Attempts           []AttemptOutcome `json:"attempts"`
}

func LoadAutofixPlan(path string) (AutofixPlan, error) {
	var c AutofixPlan
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.Repo != "" && !filepath.IsAbs(c.Repo) {
		c.Repo = filepath.Join(filepath.Dir(path), c.Repo)
	}
	if c.ArtifactsDir != "" && !filepath.IsAbs(c.ArtifactsDir) {
		c.ArtifactsDir = filepath.Join(filepath.Dir(path), c.ArtifactsDir)
	}
	return c, nil
}

func ValidateAutofixPlan(c AutofixPlan) []string {
	errs := []string{}
	if c.Objective == "" {
		errs = append(errs, "objective is required")
	}
	if c.Repo == "" {
		errs = append(errs, "repo is required")
	}
	if c.ArtifactsDir == "" {
		errs = append(errs, "artifactsDir is required")
	}
	if len(c.Attempts) == 0 {
		errs = append(errs, "at least one attempt is required")
	}
	for i, attempt := range c.Attempts {
		attempt.Repo = c.Repo
		if attempt.ArtifactsDir == "" {
			attempt.ArtifactsDir = filepath.Join(c.ArtifactsDir, fmt.Sprintf("attempt-%d", i+1))
		}
		if sub := ValidateAttemptFile(attempt); len(sub) > 0 {
			errs = append(errs, fmt.Sprintf("attempts[%d]: %s", i, stringsJoin(sub)))
		}
	}
	return errs
}

func rankAttemptsByLessons(attempts []AttemptFile, artifactsDir string, fingerprintHints []string) []AttemptFile {
	scores := map[string]int{}
	attemptKinds := map[string]string{}
	for _, attempt := range attempts {
		attemptKinds[attempt.Attempt.Name] = MutationKind(attempt.Mutation, attempt.Attempt)
	}
	hintSet := map[string]struct{}{}
	for _, h := range fingerprintHints {
		hintSet[h] = struct{}{}
	}
	_ = filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Name() != "lessons.jsonl" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range stringsSplitLines(string(data)) {
			if line == "" {
				continue
			}
			var lesson LessonRecord
			if json.Unmarshal([]byte(line), &lesson) == nil {
				matchedFingerprint := len(hintSet) == 0
				for _, fp := range lesson.Fingerprints {
					if _, ok := hintSet[fp.Signature]; ok {
						matchedFingerprint = true
						break
					}
				}
				if lesson.Success {
					scores[lesson.AttemptName] += 10
					if matchedFingerprint {
						for name, kind := range attemptKinds {
							if kind == lesson.MutationKind {
								scores[name] += 5
							}
						}
					}
				} else {
					scores[lesson.AttemptName] -= 1
				}
			}
		}
		return nil
	})
	out := append([]AttemptFile(nil), attempts...)
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := out[i].Attempt.Name, out[j].Attempt.Name
		if scores[ai] == scores[aj] {
			return ai < aj
		}
		return scores[ai] > scores[aj]
	})
	return out
}

func stringsSplitLines(s string) []string {
	lines := []string{}
	start := 0
	for i, ch := range s {
		if ch == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func RunAutofixPlan(plan AutofixPlan) (string, error) {
	if err := os.MkdirAll(plan.ArtifactsDir, 0o755); err != nil {
		return "", err
	}
	checkpoint := plan.Checkpoint
	if checkpoint == "" {
		sha, err := GitHeadSHA(plan.Repo)
		if err != nil {
			return "", err
		}
		checkpoint = sha
	}
	started := time.Now().UTC()
	attempts := rankAttemptsByLessons(plan.Attempts, plan.ArtifactsDir, plan.FingerprintHints)
	summary := AutofixSummary{
		Objective: plan.Objective,
		Repo:      plan.Repo,
		StartedAt: started.Format(time.RFC3339),
		Attempts:  []AttemptOutcome{},
	}
	for i, attempt := range attempts {
		attempt.Repo = plan.Repo
		attempt.Checkpoint = checkpoint
		if attempt.ArtifactsDir == "" {
			attempt.ArtifactsDir = filepath.Join(plan.ArtifactsDir, fmt.Sprintf("attempt-%d", i+1))
		}
		outcome, err := RunAttemptFile(attempt)
		if err != nil {
			return "", err
		}
		summary.Attempts = append(summary.Attempts, outcome)
		if outcome.Success {
			summary.Success = true
			summary.WinningAttempt = outcome.Name
			promotedSHA, promoted, err := PromoteWinningAttempt(plan.Repo, outcome.Name)
			if err != nil {
				return "", err
			}
			summary.WinnerPromoted = promoted
			summary.PromotedCheckpoint = promotedSHA
			break
		}
	}
	summary.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	summaryPath := filepath.Join(plan.ArtifactsDir, fmt.Sprintf("autofix-summary-%d.json", started.Unix()))
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(summaryPath, data, 0o644); err != nil {
		return "", err
	}
	if !summary.Success {
		return summaryPath, fmt.Errorf("autofix failed: no attempt validated")
	}
	return summaryPath, nil
}
