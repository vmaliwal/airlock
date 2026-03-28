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
