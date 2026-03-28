package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRankAttemptsByLessonsPrefersPriorSuccess(t *testing.T) {
	dir, err := os.MkdirTemp("", "airlock-autofix-rank-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	attemptDir := filepath.Join(dir, "attempt-1")
	if err := os.MkdirAll(attemptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attemptDir, "lessons.jsonl"), []byte(`{"attemptName":"winner","success":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	attempts := []AttemptFile{{Attempt: AttemptSpec{Name: "loser"}}, {Attempt: AttemptSpec{Name: "winner"}}}
	ranked := rankAttemptsByLessons(attempts, dir, nil)
	if ranked[0].Attempt.Name != "winner" {
		t.Fatalf("expected winner first, got %#v", ranked)
	}
}
