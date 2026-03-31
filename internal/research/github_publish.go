package research

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmaliwal/airlock/internal/util"
)

const (
	GitHubCreateDraftPREnv = "AIRLOCK_GITHUB_CREATE_DRAFT_PR"
	GitHubAPIBaseURLEnv    = "AIRLOCK_GITHUB_API_BASE_URL"
)

type DraftPRPublication struct {
	URL             string `json:"url,omitempty"`
	Number          int    `json:"number,omitempty"`
	Branch          string `json:"branch,omitempty"`
	BaseBranch      string `json:"base_branch,omitempty"`
	IssueCommentURL string `json:"issue_comment_url,omitempty"`
}

func GitHubDraftPRPublishingEnabled() bool {
	return strings.TrimSpace(os.Getenv(GitHubCreateDraftPREnv)) == "1"
}

func CreateDraftPRFromFix(result FixResult, baseBranch string) (DraftPRPublication, error) {
	if strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) == "" {
		return DraftPRPublication{}, fmt.Errorf("GITHUB_TOKEN is required for draft PR creation")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return DraftPRPublication{}, fmt.Errorf("base branch is required for draft PR creation")
	}
	branch := draftPRBranchName(result.Issue)
	if err := GitCheckoutBranch(result.RepoPath, branch, true); err != nil {
		return DraftPRPublication{}, err
	}
	if err := GitPushBranchToGitHub(result.RepoPath, branch, os.Getenv("GITHUB_TOKEN")); err != nil {
		return DraftPRPublication{}, err
	}
	bodyPath := result.DraftPRPath
	if strings.TrimSpace(bodyPath) == "" {
		return DraftPRPublication{}, fmt.Errorf("draft PR artifact path is required")
	}
	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		return DraftPRPublication{}, err
	}
	title, body := splitDraftPRBody(string(bodyBytes), result)
	pub, err := createGitHubDraftPR(result.Issue, branch, baseBranch, title, body)
	if err != nil {
		return DraftPRPublication{}, err
	}
	if commentURL, err := createGitHubIssueComment(result.Issue, renderIssueComment(result, pub)); err == nil {
		pub.IssueCommentURL = commentURL
	}
	return pub, nil
}

func createGitHubDraftPR(issue GitHubIssue, head, base, title, body string) (DraftPRPublication, error) {
	payload := map[string]any{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
		"draft": true,
	}
	data, _ := json.Marshal(payload)
	apiBase := strings.TrimSpace(os.Getenv(GitHubAPIBaseURLEnv))
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", strings.TrimRight(apiBase, "/"), issue.Owner, issue.Repo)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return DraftPRPublication{}, err
	}
	req.Header.Set("User-Agent", "airlock")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("GITHUB_TOKEN")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return DraftPRPublication{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return DraftPRPublication{}, fmt.Errorf("github draft pr create failed: %s: %s", resp.Status, strings.TrimSpace(buf.String()))
	}
	var out struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return DraftPRPublication{}, err
	}
	return DraftPRPublication{URL: out.HTMLURL, Number: out.Number, Branch: head, BaseBranch: base}, nil
}

func createGitHubIssueComment(issue GitHubIssue, body string) (string, error) {
	payload := map[string]any{"body": body}
	data, _ := json.Marshal(payload)
	apiBase := strings.TrimSpace(os.Getenv(GitHubAPIBaseURLEnv))
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", strings.TrimRight(apiBase, "/"), issue.Owner, issue.Repo, issue.Number)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "airlock")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("GITHUB_TOKEN")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return "", fmt.Errorf("github issue comment create failed: %s: %s", resp.Status, strings.TrimSpace(buf.String()))
	}
	var out struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.HTMLURL, nil
}

func renderIssueComment(result FixResult, pub DraftPRPublication) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Airlock opened a draft PR: %s\n\n", pub.URL)
	if result.ReviewPacketPath != "" {
		fmt.Fprintf(&b, "- Review packet: `%s`\n", result.ReviewPacketPath)
	}
	if result.DraftPRPath != "" {
		fmt.Fprintf(&b, "- Draft PR artifact: `%s`\n", result.DraftPRPath)
	}
	return b.String()
}

func splitDraftPRBody(raw string, result FixResult) (string, string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	parts := strings.SplitN(raw, "\n", 2)
	title := strings.TrimSpace(parts[0])
	if title == "" {
		title = strings.TrimSpace(result.Issue.Title)
	}
	if title == "" {
		title = fmt.Sprintf("fix issue #%d", result.Issue.Number)
	}
	body := ""
	if len(parts) > 1 {
		body = strings.TrimLeft(parts[1], "\n")
	}
	return title, body
}

func draftPRBranchName(issue GitHubIssue) string {
	slug := util.SafeName(issue.Title)
	if slug == "" {
		slug = "fix"
	}
	return fmt.Sprintf("airlock/issue-%d-%s", issue.Number, slug)
}

func GitCurrentBranch(repo string) (string, error) {
	out, err := util.RunLocal("git", []string{"rev-parse", "--abbrev-ref", "HEAD"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GitCheckoutBranch(repo, branch string, reset bool) error {
	args := []string{"checkout"}
	if reset {
		args = append(args, "-B")
	} else {
		args = append(args, "-b")
	}
	args = append(args, branch)
	_, err := util.RunLocal("git", args, util.RunOptions{Cwd: repo})
	return err
}

func GitPushBranchToGitHub(repo, branch, token string) error {
	remote, err := GitRemoteOrigin(repo)
	if err != nil {
		return err
	}
	remote = NormalizeCloneURL(remote)
	if !strings.HasPrefix(remote, "https://github.com/") {
		return fmt.Errorf("github push only supports https github remotes today: %s", remote)
	}
	askpassPath, env, err := gitHubAskpassEnv(token)
	if err != nil {
		return err
	}
	defer os.Remove(askpassPath)
	_, err = util.RunLocal("git", []string{"push", remote, "HEAD:refs/heads/" + branch}, util.RunOptions{Cwd: repo, Env: env, Timeout: 20 * time.Minute})
	return err
}

func gitHubAskpassEnv(token string) (string, []string, error) {
	dir, err := os.MkdirTemp("", "airlock-gh-askpass-")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "askpass.sh")
	script := "#!/usr/bin/env bash\nset -euo pipefail\ncase \"${1:-}\" in\n  *Username*) printf '%s\\n' 'x-access-token' ;;\n  *) printf '%s\\n' \"${AIRLOCK_GITHUB_TOKEN:-}\" ;;\nesac\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return "", nil, err
	}
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS="+path, "AIRLOCK_GITHUB_TOKEN="+token)
	return path, env, nil
}
