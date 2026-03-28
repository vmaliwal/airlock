package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/util"
)

type CampaignEntry struct {
	Name     string `json:"name"`
	Contract string `json:"contract"`
}

type CampaignPlanSuccess struct {
	MaxFailed *int `json:"max_failed,omitempty"`
}

type CampaignPlan struct {
	Objective     string              `json:"objective"`
	ArtifactsDir  string              `json:"artifactsDir"`
	Entries       []CampaignEntry     `json:"entries"`
	StopOnFailure bool                `json:"stop_on_failure,omitempty"`
	Success       CampaignPlanSuccess `json:"success,omitempty"`
}

type CampaignEntryResult struct {
	Name        string `json:"name"`
	Contract    string `json:"contract"`
	SummaryPath string `json:"summaryPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

type CampaignRunSummary struct {
	Objective     string                `json:"objective"`
	StartedAt     string                `json:"startedAt"`
	FinishedAt    string                `json:"finishedAt"`
	Success       bool                  `json:"success"`
	TotalEntries  int                   `json:"totalEntries"`
	PassedEntries int                   `json:"passedEntries"`
	FailedEntries int                   `json:"failedEntries"`
	Entries       []CampaignEntryResult `json:"entries"`
}

func LoadCampaignPlan(path string) (CampaignPlan, error) {
	var c CampaignPlan
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func ValidateCampaignPlan(c CampaignPlan) []string {
	errs := []string{}
	if c.Objective == "" {
		errs = append(errs, "objective is required")
	}
	if c.ArtifactsDir == "" {
		errs = append(errs, "artifactsDir is required")
	}
	if len(c.Entries) == 0 {
		errs = append(errs, "at least one campaign entry is required")
	}
	for i, entry := range c.Entries {
		if entry.Name == "" {
			errs = append(errs, fmt.Sprintf("entries[%d].name is required", i))
		}
		if entry.Contract == "" {
			errs = append(errs, fmt.Sprintf("entries[%d].contract is required", i))
		}
	}
	return errs
}

func RunCampaignPlan(planPath string, plan CampaignPlan) (string, error) {
	if err := util.EnsureDir(plan.ArtifactsDir); err != nil {
		return "", err
	}
	started := time.Now().UTC()
	baseDir := filepath.Dir(planPath)
	results := make([]CampaignEntryResult, 0, len(plan.Entries))
	passed := 0
	failed := 0

	lessonsPath := filepath.Join(plan.ArtifactsDir, "campaign-lessons.jsonl")
	for _, entry := range plan.Entries {
		resolved := entry.Contract
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(baseDir, resolved)
		}
		rc, err := LoadRunContract(resolved)
		if err != nil {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, Success: false, Error: err.Error()})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		if errs := ValidateRunContract(rc); len(errs) > 0 {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, Success: false, Error: stringsJoin(errs)})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		compiled, err := CompileRunContract(rc)
		if err != nil {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, Success: false, Error: err.Error()})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		summaryPath, err := ExecuteCompiledContract(compiled)
		if err != nil {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, Success: false, Error: err.Error()})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		summaryData, err := os.ReadFile(summaryPath)
		if err != nil {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, SummaryPath: summaryPath, Success: false, Error: err.Error()})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		var summary contract.RunSummary
		if err := json.Unmarshal(summaryData, &summary); err != nil {
			failed++
			results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, SummaryPath: summaryPath, Success: false, Error: err.Error()})
			_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: false})
			if plan.StopOnFailure {
				break
			}
			continue
		}
		if summary.Success {
			passed++
		} else {
			failed++
		}
		results = append(results, CampaignEntryResult{Name: entry.Name, Contract: resolved, SummaryPath: summaryPath, Success: summary.Success})
		_ = AppendLesson(lessonsPath, LessonRecord{Timestamp: time.Now().UTC().Format(time.RFC3339), Repo: resolved, AttemptName: entry.Name, Success: summary.Success})
		if plan.StopOnFailure && !summary.Success {
			break
		}
	}

	success := failed == 0
	if plan.Success.MaxFailed != nil {
		success = failed <= *plan.Success.MaxFailed
	}

	summary := CampaignRunSummary{
		Objective:     plan.Objective,
		StartedAt:     started.Format(time.RFC3339),
		FinishedAt:    time.Now().UTC().Format(time.RFC3339),
		Success:       success,
		TotalEntries:  len(plan.Entries),
		PassedEntries: passed,
		FailedEntries: failed,
		Entries:       results,
	}

	summaryPath := filepath.Join(plan.ArtifactsDir, fmt.Sprintf("campaign-summary-%d.json", started.Unix()))
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(summaryPath, data, 0o644); err != nil {
		return "", err
	}
	if !success {
		return summaryPath, fmt.Errorf("campaign failed: %d/%d entries failed", failed, len(plan.Entries))
	}
	return summaryPath, nil
}

func stringsJoin(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for _, item := range items[1:] {
		out += "; " + item
	}
	return out
}
