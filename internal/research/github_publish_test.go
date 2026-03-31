package research

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestSplitDraftPRBody(t *testing.T) {
	title, body := splitDraftPRBody("fix parser edge case\n\n## Summary\nhello", FixResult{Issue: GitHubIssue{Number: 123}})
	if title != "fix parser edge case" {
		t.Fatalf("unexpected title: %q", title)
	}
	if !strings.Contains(body, "## Summary") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestCreateGitHubDraftPR(t *testing.T) {
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldBase := os.Getenv(GitHubAPIBaseURLEnv)
	defer os.Setenv("GITHUB_TOKEN", oldToken)
	defer os.Setenv(GitHubAPIBaseURLEnv, oldBase)
	os.Setenv("GITHUB_TOKEN", "ghp_test")
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer ghp_test" {
			t.Fatalf("unexpected auth header: %q", gotAuth)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"html_url": "https://github.com/owner/repo/pull/10", "number": 10})
	}))
	defer srv.Close()
	os.Setenv(GitHubAPIBaseURLEnv, srv.URL)
	pub, err := createGitHubDraftPR(GitHubIssue{Owner: "owner", Repo: "repo"}, "airlock/issue-1-fix", "main", "fix parser", "body")
	if err != nil {
		t.Fatal(err)
	}
	if pub.URL != "https://github.com/owner/repo/pull/10" || pub.Number != 10 {
		t.Fatalf("unexpected publication: %#v", pub)
	}
	if got["draft"] != true || got["head"] != "airlock/issue-1-fix" || got["base"] != "main" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestGitPushBranchToGitHubRejectsNonGitHubRemote(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "hello\n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "https://example.com/acme/repo.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	if err := GitPushBranchToGitHub(repo, "airlock/issue-1-fix", "token"); err == nil {
		t.Fatal("expected github remote restriction error")
	}
}

func TestGitHubAskpassEnv(t *testing.T) {
	path, env, err := gitHubAskpassEnv("secret")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "GIT_ASKPASS=") || !strings.Contains(joined, "AIRLOCK_GITHUB_TOKEN=secret") {
		t.Fatalf("unexpected env: %s", joined)
	}
}
