package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const LessonsRootEnv = "AIRLOCK_LESSONS_ROOT"

type MutationKindScore struct {
	Kind    string   `json:"kind"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

type PlanReport struct {
	Input                PlanInput           `json:"input"`
	Investigation        InvestigationReport `json:"investigation"`
	RankedMutationKinds  []MutationKindScore `json:"rankedMutationKinds,omitempty"`
	CandidateActionKinds []string            `json:"candidateActionKinds,omitempty"`
	LessonsSearchRoots   []string            `json:"lessonsSearchRoots,omitempty"`
	CandidateCommands    []string            `json:"candidateCommands,omitempty"`
}

type loadedLesson struct {
	Lesson LessonRecord
	Path   string
}

func PlanRepo(path string, vmBackend string, allowHostExecution bool) (PlanReport, error) {
	return PlanFromInput(PlanInput{RepoPath: path}, vmBackend, allowHostExecution)
}

func PlanFromInput(input PlanInput, vmBackend string, allowHostExecution bool) (PlanReport, error) {
	investigation, err := InvestigateRepo(input.RepoPath, vmBackend, allowHostExecution)
	if err != nil {
		return PlanReport{}, err
	}
	roots := lessonSearchRoots(investigation.Profile.RepoRoot)
	lessons := loadLessonsFromRoots(roots)
	ranked := rankMutationKinds(investigation.Profile, lessons)
	actionKinds := candidateActionKinds(ranked)
	candidateCommands := rankedCommands(input, investigation)
	return PlanReport{
		Input:                input,
		Investigation:        investigation,
		RankedMutationKinds:  ranked,
		CandidateActionKinds: actionKinds,
		LessonsSearchRoots:   roots,
		CandidateCommands:    candidateCommands,
	}, nil
}

func lessonSearchRoots(repoRoot string) []string {
	roots := []string{}
	if v := os.Getenv(LessonsRootEnv); v != "" {
		roots = append(roots, v)
	}
	if repoRoot != "" {
		roots = append(roots, repoRoot)
		roots = append(roots, filepath.Dir(repoRoot))
	}
	return dedupeStrings(roots)
}

func loadLessonsFromRoots(roots []string) []loadedLesson {
	var out []loadedLesson
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
					out = append(out, loadedLesson{Lesson: lesson, Path: path})
				}
			}
			return nil
		})
	}
	return out
}

func rankMutationKinds(profile RepoProfile, lessons []loadedLesson) []MutationKindScore {
	scores := map[string]int{}
	reasons := map[string][]string{}
	for _, kind := range defaultMutationKinds(profile.RepoType) {
		scores[kind] += 1
		reasons[kind] = append(reasons[kind], fmt.Sprintf("repo_type_default:%s", profile.RepoType))
	}
	for _, item := range lessons {
		kind := item.Lesson.MutationKind
		if kind == "" || kind == "unknown" {
			continue
		}
		weight := 0
		if item.Lesson.Success {
			weight += 5
		} else {
			weight -= 1
		}
		if samePath(item.Lesson.Repo, profile.RepoRoot) || samePath(item.Lesson.Repo, profile.RepoPath) {
			weight += 5
			reasons[kind] = append(reasons[kind], "same_repo_lesson")
		}
		scores[kind] += weight
		if item.Lesson.Success {
			reasons[kind] = append(reasons[kind], "prior_success")
		} else {
			reasons[kind] = append(reasons[kind], "prior_failure")
		}
	}
	var ranked []MutationKindScore
	for kind, score := range scores {
		ranked = append(ranked, MutationKindScore{Kind: kind, Score: score, Reasons: dedupeStrings(reasons[kind])})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Kind < ranked[j].Kind
		}
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

func defaultMutationKinds(repoType string) []string {
	switch repoType {
	case "go":
		return []string{"replace_line", "search_replace", "insert_after", "apply_patch"}
	case "python":
		return []string{"search_replace", "replace_line", "create_file", "apply_patch"}
	case "node":
		return []string{"search_replace", "replace_line", "insert_after", "apply_patch"}
	default:
		return []string{"search_replace", "replace_line", "apply_patch"}
	}
}

func candidateActionKinds(ranked []MutationKindScore) []string {
	out := []string{}
	for i, item := range ranked {
		if i >= 4 {
			break
		}
		switch item.Kind {
		case "apply_patch", "search_replace", "replace_line", "insert_after", "create_file":
			out = append(out, item.Kind)
		}
	}
	return dedupeStrings(out)
}

func rankedCommands(input PlanInput, investigation InvestigationReport) []string {
	out := []string{}
	if input.FailingCommand != "" {
		out = append(out, input.FailingCommand)
	}
	out = append(out, investigation.CandidateReproduction...)
	out = append(out, investigation.CandidateValidation...)
	if input.IssueURL != "" {
		out = append(out, "issue_context="+input.IssueURL)
	}
	if input.FailureText != "" {
		out = append(out, "failure_text_present")
	}
	return dedupeStrings(out)
}
