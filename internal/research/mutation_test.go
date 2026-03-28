package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyMutationSpecInsertAfter(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-mutation-insert-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	path := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyMutationSpec(repo, MutationSpec{InsertAfter: &InsertAfterMutation{Path: "a.txt", AnchorText: "one\n", InsertText: "inserted\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", res)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "one\ninserted\ntwo\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestApplyMutationSpecReplaceLine(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-mutation-line-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	path := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyMutationSpec(repo, MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: "a.txt", OldLine: "two", NewLine: "three"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", res)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "one\nthree\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestApplyMutationSpecCreateFile(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-mutation-create-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	res, err := ApplyMutationSpec(repo, MutationSpec{CreateFile: &CreateFileMutation{Path: "dir/a.txt", Content: "hello\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", res)
	}
	data, _ := os.ReadFile(filepath.Join(repo, "dir", "a.txt"))
	if string(data) != "hello\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestApplyMutationSpecApplyPatch(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-mutation-patch-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "one\n"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "a.txt")
	patch := "diff --git a/a.txt b/a.txt\nindex 5626abf..814f4a4 100644\n--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-one\n+two\n"
	res, err := ApplyMutationSpec(repo, MutationSpec{ApplyPatch: &ApplyPatchMutation{Patch: patch}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", res)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "two\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestApplyMutationSpecEnsureLine(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-mutation-ensure-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	path := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyMutationSpec(repo, MutationSpec{EnsureLine: &EnsureLineMutation{Path: "a.txt", Line: "two"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", res)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "one\ntwo\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}
