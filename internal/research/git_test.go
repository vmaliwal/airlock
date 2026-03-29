package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitCommitAllNoopOnCleanTree(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-git-clean-commit-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "hello\n"}); err != nil {
		t.Fatal(err)
	}
	before, err := GitHeadSHA(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := GitCommitAll(repo, "noop commit"); err != nil {
		t.Fatal(err)
	}
	after, err := GitHeadSHA(repo)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("expected no-op commit to keep HEAD unchanged: %s != %s", before, after)
	}
}

func TestGitHelpersLifecycle(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-git-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)

	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "hello\n"}); err != nil {
		t.Fatal(err)
	}
	sha, err := GitHeadSHA(repo)
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("expected head sha")
	}
	dirty, err := GitIsDirty(repo)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("expected clean repo")
	}

	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err = GitIsDirty(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("expected dirty repo")
	}
	stats, err := GitDiffNumstat(repo)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FilesChangedCount != 1 {
		t.Fatalf("expected one tracked changed file before add, got %d", stats.FilesChangedCount)
	}
	patchPath := filepath.Join(repo, "out.patch")
	if err := GitWritePatch(repo, patchPath); err != nil {
		t.Fatal(err)
	}
	patch, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patch), "a.txt") {
		t.Fatalf("expected patch to mention a.txt, got %s", string(patch))
	}
	if err := GitResetAttempt(repo); err != nil {
		t.Fatal(err)
	}
	dirty, err = GitIsDirty(repo)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("expected clean after reset")
	}
	if _, err := os.Stat(filepath.Join(repo, "b.txt")); !os.IsNotExist(err) {
		t.Fatal("expected untracked file to be removed")
	}
}
