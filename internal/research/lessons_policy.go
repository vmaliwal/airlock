package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type LessonPolicyScore struct {
	Kind    string   `json:"kind"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

func rankMutationKindsWithContext(profile RepoProfile, lessons []loadedLesson, hints []string) []MutationKindScore {
	scores := map[string]int{}
	reasons := map[string][]string{}
	hintSet := map[string]struct{}{}
	for _, h := range hints {
		hintSet[h] = struct{}{}
	}
	for _, kind := range defaultMutationKinds(profile.RepoType) {
		scores[kind] += 1
		reasons[kind] = append(reasons[kind], "repo_type_default:"+profile.RepoType)
	}
	for _, kind := range fingerprintDefaultKinds(hints) {
		scores[kind] += 3
		reasons[kind] = append(reasons[kind], "fingerprint_default")
	}
	for _, item := range lessons {
		kind := item.Lesson.MutationKind
		if kind == "" || kind == "unknown" {
			continue
		}
		weight := 0
		matchedFingerprint := len(hintSet) == 0
		for _, fp := range item.Lesson.Fingerprints {
			if _, ok := hintSet[fp.Signature]; ok {
				matchedFingerprint = true
				break
			}
		}
		if item.Lesson.Success {
			weight += 5
			reasons[kind] = append(reasons[kind], "prior_success")
		} else {
			weight -= 1
			reasons[kind] = append(reasons[kind], "prior_failure")
		}
		if samePath(item.Lesson.Repo, profile.RepoRoot) || samePath(item.Lesson.Repo, profile.RepoPath) {
			weight += 5
			reasons[kind] = append(reasons[kind], "same_repo_lesson")
		}
		if matchedFingerprint {
			weight += 4
			reasons[kind] = append(reasons[kind], "matching_fingerprint_lesson")
		}
		scores[kind] += weight
	}
	out := make([]MutationKindScore, 0, len(scores))
	for kind, score := range scores {
		out = append(out, MutationKindScore{Kind: kind, Score: score, Reasons: dedupeStrings(reasons[kind])})
	}
	sortMutationScores(out)
	return out
}

func fingerprintDefaultKinds(hints []string) []string {
	kinds := []string{}
	for _, h := range hints {
		switch {
		case strings.HasPrefix(h, "panic:"), strings.Contains(h, "nil pointer dereference"):
			kinds = append(kinds, "nil_guard", "error_return")
		case strings.HasPrefix(h, "package_failure:"), strings.HasPrefix(h, "test_failure:"):
			kinds = append(kinds, "replace_line", "search_replace", "apply_patch")
		case strings.HasPrefix(h, "command_failure:"):
			kinds = append(kinds, "ensure_line", "create_file", "apply_patch")
		}
	}
	return dedupeStrings(kinds)
}

func sortMutationScores(out []MutationKindScore) {
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[i].Score || (out[j].Score == out[i].Score && out[j].Kind < out[i].Kind) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
}

func collectFingerprintHintsFromFailureText(failureText string) []string {
	if strings.TrimSpace(failureText) == "" {
		return nil
	}
	fps := ExtractFailureSignatures(CommandResult{Command: "failure_text", ExitCode: 1, Stderr: failureText})
	return dedupeStrings(fps)
}

func maybeLoadLessonsFile(path string) []loadedLesson {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := []loadedLesson{}
	for _, line := range stringsSplitLines(string(data)) {
		if line == "" {
			continue
		}
		var lesson LessonRecord
		if json.Unmarshal([]byte(line), &lesson) == nil {
			out = append(out, loadedLesson{Lesson: lesson, Path: path})
		}
	}
	return out
}

func loadLessonsFromRoots(roots []string) []loadedLesson {
	var out []loadedLesson
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || info.Name() != "lessons.jsonl" {
				return nil
			}
			out = append(out, maybeLoadLessonsFile(path)...)
			return nil
		})
	}
	return out
}
