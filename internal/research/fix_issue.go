package research

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type GitHubIssue struct {
	URL        string   `json:"url"`
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	State      string   `json:"state"`
	HTMLURL    string   `json:"html_url"`
	CloneURL   string   `json:"clone_url"`
	Labels     []string `json:"labels,omitempty"`
	DefaultRef string   `json:"default_ref,omitempty"`
}

type FixProgressEvent struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Done    bool   `json:"done,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

type FixResult struct {
	Issue                  GitHubIssue     `json:"issue"`
	RepoPath               string          `json:"repoPath"`
	PlanInput              PlanInput       `json:"planInput"`
	ReadonlySummaryPath    string          `json:"readonlySummaryPath,omitempty"`
	ReproductionResults    map[string]any  `json:"reproductionResults,omitempty"`
	Synthesis              SynthesisReport `json:"synthesis"`
	AutofixContractSummary string          `json:"autofixContractSummary,omitempty"`
	AutofixResult          map[string]any  `json:"autofixResult,omitempty"`
}

func ResolveGitHubIssue(issueURL string) (GitHubIssue, error) {
	owner, repo, num, err := ParseGitHubIssueURL(issueURL)
	if err != nil {
		return GitHubIssue{}, err
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, num), nil)
	if err != nil {
		return GitHubIssue{}, err
	}
	req.Header.Set("User-Agent", "airlock")
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return GitHubIssue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubIssue{}, fmt.Errorf("github issue fetch failed: %s", resp.Status)
	}
	var payload struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		Labels  []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GitHubIssue{}, err
	}
	labels := []string{}
	for _, label := range payload.Labels {
		labels = append(labels, label.Name)
	}
	return GitHubIssue{
		URL:      issueURL,
		Owner:    owner,
		Repo:     repo,
		Number:   num,
		Title:    payload.Title,
		Body:     payload.Body,
		State:    payload.State,
		HTMLURL:  payload.HTMLURL,
		CloneURL: fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
		Labels:   labels,
	}, nil
}

func ParseGitHubIssueURL(raw string) (owner, repo string, number int, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", 0, err
	}
	if u.Host != "github.com" {
		return "", "", 0, fmt.Errorf("unsupported issue host %q", u.Host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "issues" {
		return "", "", 0, fmt.Errorf("unsupported github issue path %q", u.Path)
	}
	owner = parts[0]
	repo = parts[1]
	_, err = fmt.Sscanf(parts[3], "%d", &number)
	if err != nil {
		return "", "", 0, fmt.Errorf("parse issue number: %w", err)
	}
	return owner, repo, number, nil
}

func CloneIssueRepo(issue GitHubIssue) (string, error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("airlock-fix-%s-%s-%d-", issue.Owner, issue.Repo, issue.Number))
	if err != nil {
		return "", err
	}
	if _, err := RunLocalCommand(filepath.Dir(dir), fmt.Sprintf("git clone --depth=1 %s %s", shellEscape(issue.CloneURL), shellEscape(dir)), 20*time.Minute); err != nil {
		return "", err
	}
	return dir, nil
}

func BuildPlanInputFromIssue(issue GitHubIssue, repoPath string) PlanInput {
	failureText := strings.TrimSpace(issue.Title)
	notes := strings.TrimSpace(issue.Body)
	if len(issue.Labels) > 0 {
		notes = strings.TrimSpace(notes + "\n\nlabels: " + strings.Join(issue.Labels, ", "))
	}
	return PlanInput{
		RepoPath:       repoPath,
		IssueURL:       issue.URL,
		FailingCommand: inferFailingCommandFromIssue(issue),
		FailureText:    failureText,
		Notes:          notes,
	}
}

func inferFailingCommandFromIssue(issue GitHubIssue) string {
	body := issue.Body
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^(go test\s+.+)$`),
		regexp.MustCompile(`(?m)^(pytest\s+.+)$`),
		regexp.MustCompile(`(?m)^(python\s+-m\s+pytest\s+.+)$`),
		regexp.MustCompile(`(?m)^(npm test(?:\s+--\s+.+)?)$`),
		regexp.MustCompile(`(?m)^(pnpm test(?:\s+.+)?)$`),
		regexp.MustCompile(`(?m)^(yarn test(?:\s+.+)?)$`),
		regexp.MustCompile(`(?m)^(cargo test\s+.+)$`),
	}
	for _, re := range patterns {
		if m := re.FindStringSubmatch(body); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func ReadSiblingArtifact(summaryPath, suffix string) (map[string]any, bool) {
	base := strings.TrimSuffix(summaryPath, "-summary.json")
	path := base + "-" + suffix
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	return out, true
}
