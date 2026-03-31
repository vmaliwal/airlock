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
	Issue                  GitHubIssue         `json:"issue"`
	RepoPath               string              `json:"repoPath"`
	PlanInput              PlanInput           `json:"planInput"`
	IssueSignals           IssueSignals        `json:"issueSignals,omitempty"`
	ReadonlySummaryPath    string              `json:"readonlySummaryPath,omitempty"`
	ReproductionResults    map[string]any      `json:"reproductionResults,omitempty"`
	Synthesis              SynthesisReport     `json:"synthesis"`
	AutofixContractSummary string              `json:"autofixContractSummary,omitempty"`
	AutofixResult          map[string]any      `json:"autofixResult,omitempty"`
	FixLoop                AutofixLoopSummary  `json:"fixLoop,omitempty"`
	ReviewPacketPath       string              `json:"reviewPacketPath,omitempty"`
	DraftPRPath            string              `json:"draftPRPath,omitempty"`
	DraftPRPublication     *DraftPRPublication `json:"draftPRPublication,omitempty"`
}

// IssueSignals captures lightweight classification signals derived from the
// issue body before any execution occurs. Used for honest early classification.
type IssueSignals struct {
	ServiceDependent    bool   `json:"service_dependent,omitempty"`
	InferredCommandKind string `json:"inferred_command_kind,omitempty"` // "repro", "setup", "none"
	Notes               string `json:"notes,omitempty"`
}

// ClassifyIssueSignals derives signals from an issue body and inferred command
// without cloning or executing anything.
func ClassifyIssueSignals(issue GitHubIssue, inferredCmd string) IssueSignals {
	sig := IssueSignals{}
	if IssueBodyServiceDependent(issue.Body) {
		sig.ServiceDependent = true
		sig.Notes = "issue body describes a live-service reproduction (HMR, dev server, browser interaction); offline test reproduction may not be possible"
	}
	switch {
	case inferredCmd == "":
		sig.InferredCommandKind = "none"
	case IsSetupCommand(inferredCmd):
		sig.InferredCommandKind = "setup"
		if sig.Notes == "" {
			sig.Notes = "inferred command looks like a setup/build step, not a test assertion; reproduction may be incomplete"
		} else {
			sig.Notes += "; inferred command looks like a setup/build step"
		}
	default:
		sig.InferredCommandKind = "repro"
	}
	return sig
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

// inferFailingCommandFromIssue extracts the most likely reproduction command
// from an issue body. It tries bare-line patterns first (high confidence),
// then falls back to fenced-shell-block extraction (lower confidence).
func inferFailingCommandFromIssue(issue GitHubIssue) string {
	body := issue.Body
	if cmd := inferBareLineCommand(body); cmd != "" {
		return cmd
	}
	return inferFencedBlockCommand(body)
}

// bareLinePatterns are matched against individual lines with no surrounding
// context required. Order matters — more specific patterns first.
var bareLinePatterns = []*regexp.Regexp{
	// Go
	regexp.MustCompile(`^(go test\s+\S.*)$`),
	// Python
	regexp.MustCompile(`^(python\s+-m\s+pytest\s+\S.*)$`),
	regexp.MustCompile(`^(pytest\s+\S.*)$`),
	regexp.MustCompile(`^(pytest)$`),
	regexp.MustCompile(`^(\.venv/bin/python\s+\S.*)$`),
	// uv / modern Python tooling
	regexp.MustCompile(`^(uv run\s+\S.*)$`),
	// Rust
	regexp.MustCompile(`^(cargo test\s+\S.*)$`),
	regexp.MustCompile(`^(cargo test)$`),
	// Node (npm / pnpm / yarn)
	regexp.MustCompile(`^(npm test(?:\s+--\s+.*)?)$`),
	regexp.MustCompile(`^(npm run test(?:\s+\S.*)?)$`),
	regexp.MustCompile(`^(pnpm test(?:\s+\S.*)?)$`),
	regexp.MustCompile(`^(pnpm run test(?:\s+\S.*)?)$`),
	regexp.MustCompile(`^(yarn test(?:\s+\S.*)?)$`),
	regexp.MustCompile(`^(yarn run test(?:\s+\S.*)?)$`),
	// Node script runner
	regexp.MustCompile(`^(node\s+\S+\.(?:js|mjs|cjs|ts).*)$`),
	// Task runners
	regexp.MustCompile(`^(just\s+\S.*)$`),
	regexp.MustCompile(`^(make\s+(?:test|check|spec)\S*.*)$`),
	regexp.MustCompile(`^(make\s+test)$`),
}

func inferBareLineCommand(body string) string {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		// Strip shell prompt prefixes common in issue bodies
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "$ ")
		line = strings.TrimPrefix(line, "% ")
		line = strings.TrimPrefix(line, "> ")
		line = strings.TrimSpace(line)
		for _, re := range bareLinePatterns {
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

// fencedBlockCommandRE matches ``` ... ``` blocks including language specifiers.
var fencedBlockCommandRE = regexp.MustCompile("(?s)```(?:sh|bash|shell|console|zsh|fish|text|)?\n(.*?)```")

// reproKeywords are used to score fenced blocks for repro likelihood.
var reproKeywords = []string{"repro", "test", "spec", "fail", "example", "run", "error", "bug"}

func inferFencedBlockCommand(body string) string {
	type candidate struct {
		cmd   string
		score int
	}
	var best candidate
	blocks := fencedBlockCommandRE.FindAllStringSubmatchIndex(body, -1)
	for _, loc := range blocks {
		block := body[loc[2]:loc[3]]
		// Score the surrounding context (100 chars before block) for repro keywords
		contextStart := loc[0] - 100
		if contextStart < 0 {
			contextStart = 0
		}
		context := strings.ToLower(body[contextStart:loc[0]])
		score := 0
		for _, kw := range reproKeywords {
			if strings.Contains(context, kw) {
				score++
			}
		}
		// Also score the block content itself
		blockLower := strings.ToLower(block)
		for _, kw := range reproKeywords {
			if strings.Contains(blockLower, kw) {
				score++
			}
		}
		// Extract the first matching command from this block
		cmd := inferBareLineCommand(block)
		if cmd != "" && score > best.score {
			best = candidate{cmd: cmd, score: score}
		}
	}
	return best.cmd
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
