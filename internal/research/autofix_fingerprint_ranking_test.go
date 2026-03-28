package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRankAttemptsByLessonsPrefersMutationKindForMatchingFingerprint(t *testing.T) {
	dir, err := os.MkdirTemp("", "airlock-autofix-fp-rank-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	attemptDir := filepath.Join(dir, "attempt-1")
	if err := os.MkdirAll(attemptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lesson := `{"attemptName":"old-search","mutationKind":"search_replace","success":true,"fingerprints":[{"signature":"package_failure:x","kind":"package_failure","count":1}]}` + "\n"
	if err := os.WriteFile(filepath.Join(attemptDir, "lessons.jsonl"), []byte(lesson), 0o644); err != nil {
		t.Fatal(err)
	}
	attempts := []AttemptFile{
		{Attempt: AttemptSpec{Name: "line-fix"}, Mutation: MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: "f", OldLine: "a", NewLine: "b"}}},
		{Attempt: AttemptSpec{Name: "search-fix"}, Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "f", OldText: "a", NewText: "b"}}},
	}
	ranked := rankAttemptsByLessons(attempts, dir, []string{"package_failure:x"})
	if ranked[0].Attempt.Name != "search-fix" {
		t.Fatalf("expected search-fix first, got %#v", ranked)
	}
}
