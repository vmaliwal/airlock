package research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/invopop/jsonschema"
)

const (
	PlannerProviderEnv = "AIRLOCK_PLANNER_PROVIDER"
	PlannerModelEnv    = "AIRLOCK_PLANNER_MODEL"
)

type PlannerClient interface {
	Synthesize(context.Context, PlannerRequest) (PlannerResponse, error)
}

type PlannerRequest struct {
	Input               PlanInput            `json:"input"`
	Investigation       InvestigationReport  `json:"investigation"`
	ValidationCommand   string               `json:"validationCommand"`
	RankedMutationKinds []MutationKindScore  `json:"rankedMutationKinds,omitempty"`
	AllowedMutations    []string             `json:"allowedMutations"`
	CandidateFiles      []PlannerFileContext `json:"candidateFiles,omitempty"`
}

type PlannerFileContext struct {
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
	Score   int    `json:"score,omitempty"`
}

type PlannerResponse struct {
	Summary  string                   `json:"summary" jsonschema_description:"One short sentence describing the attempted repair strategy set."`
	Attempts []PlannerAttemptProposal `json:"attempts" jsonschema_description:"Between 1 and 5 bounded candidate attempts using only the allowed mutation kinds."`
}

type PlannerAttemptProposal struct {
	Name         string `json:"name" jsonschema_description:"Short kebab-case or concise identifier for the attempt."`
	MutationKind string `json:"mutationKind" jsonschema:"enum=search_replace,enum=replace_line,enum=insert_after,enum=ensure_line,enum=nil_guard,enum=error_return"`
	Confidence   string `json:"confidence" jsonschema:"enum=high,enum=medium,enum=low"`
	Rationale    string `json:"rationale" jsonschema_description:"Why this mutation is a plausible repair for the observed failure."`
	Path         string `json:"path" jsonschema_description:"Path relative to the selected target path."`
	OldText      string `json:"oldText,omitempty"`
	NewText      string `json:"newText,omitempty"`
	OldLine      string `json:"oldLine,omitempty"`
	NewLine      string `json:"newLine,omitempty"`
	AnchorText   string `json:"anchorText,omitempty"`
	InsertText   string `json:"insertText,omitempty"`
	Line         string `json:"line,omitempty"`
	GuardLine    string `json:"guardLine,omitempty"`
	ReturnLine   string `json:"returnLine,omitempty"`
	InsertAfter  string `json:"insertAfter,omitempty"`
}

type AnthropicPlanner struct {
	client anthropic.Client
	model  anthropic.Model
}

var plannerFactory = defaultPlannerFactory

func defaultPlannerFactory() (PlannerClient, bool, error) {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(PlannerProviderEnv))) {
	case "", "none":
		return nil, false, nil
	case "anthropic":
		apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
		if apiKey == "" {
			return nil, false, fmt.Errorf("%s=anthropic requires ANTHROPIC_API_KEY", PlannerProviderEnv)
		}
		model := strings.TrimSpace(os.Getenv(PlannerModelEnv))
		if model == "" {
			model = "claude-sonnet-4-5"
		}
		return AnthropicPlanner{
			client: anthropic.NewClient(option.WithAPIKey(apiKey)),
			model:  anthropic.Model(model),
		}, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported planner provider %q", os.Getenv(PlannerProviderEnv))
	}
}

func plannerRequestFor(input PlanInput, report PlanReport, validationCmd string) PlannerRequest {
	allowed := []string{"search_replace", "replace_line", "insert_after", "ensure_line"}
	if report.Investigation.Profile.RepoType == "go" {
		allowed = append(allowed, "nil_guard", "error_return")
	}
	avoid := avoidMutationKinds(input)
	return PlannerRequest{
		Input:               input,
		Investigation:       report.Investigation,
		ValidationCommand:   validationCmd,
		RankedMutationKinds: filterRankedMutationKinds(report.RankedMutationKinds, avoid),
		AllowedMutations:    filterAllowedMutations(allowed, avoid),
		CandidateFiles:      collectPlannerFileContext(report.Investigation.Profile.TargetPath, input),
	}
}

func collectPlannerFileContext(root string, input PlanInput) []PlannerFileContext {
	if root == "" {
		return nil
	}
	tokens := plannerSearchTokens(input.FailureText + "\n" + input.Notes + "\n" + input.FailingCommand)
	type scoredFile struct {
		path    string
		snippet string
		score   int
	}
	scored := []scoredFile{}
	seen := map[string]struct{}{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !isPlannerSourceFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		score, snippet := scorePlannerFile(rel, content, tokens)
		if score == 0 && len(scored) >= 16 {
			return nil
		}
		if snippet == "" {
			snippet = firstPlannerSnippet(content)
		}
		scored = append(scored, scoredFile{path: rel, snippet: snippet, score: score})
		seen[rel] = struct{}{}
		return nil
	})
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].path < scored[j].path
		}
		return scored[i].score > scored[j].score
	})
	out := []PlannerFileContext{}
	add := func(path string, score int) {
		if _, ok := seen[path]; !ok {
			return
		}
		for _, existing := range out {
			if existing.Path == path {
				return
			}
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return
		}
		out = append(out, PlannerFileContext{Path: path, Snippet: firstPlannerSnippet(string(data)), Score: score})
	}
	for _, item := range scored {
		if len(out) >= 6 {
			break
		}
		out = append(out, PlannerFileContext{Path: item.path, Snippet: item.snippet, Score: item.score})
		for _, paired := range pairedPlannerPaths(item.path) {
			add(paired, item.score-1)
			if len(out) >= 6 {
				break
			}
		}
	}
	return out
}

func plannerSearchTokens(s string) []string {
	replacer := strings.NewReplacer("\n", " ", "\t", " ", "(", " ", ")", " ", ":", " ", ",", " ", ".", " ", "\"", " ", "'", " ", "`", " ", "[", " ", "]", " ", "{", " ", "}", " ", "=", " ", "/", " ")
	s = replacer.Replace(s)
	parts := strings.Fields(strings.ToLower(s))
	out := []string{}
	for _, p := range parts {
		if len(p) < 4 {
			continue
		}
		if p == "expected" || p == "error" || p == "failed" || p == "failure" || p == "test" {
			continue
		}
		out = append(out, p)
	}
	symbolRE := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]{2,}`)
	for _, sym := range symbolRE.FindAllString(s, -1) {
		if len(sym) >= 4 {
			out = append(out, strings.ToLower(sym), sym)
		}
	}
	return dedupeStrings(out)
}

func isPlannerSourceFile(path string) bool {
	for _, ext := range []string{".go", ".py", ".ts", ".js", ".java", ".rs"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func scorePlannerFile(path, content string, tokens []string) (int, string) {
	lower := strings.ToLower(content)
	lowerPath := strings.ToLower(path)
	bestIdx := -1
	score := 0
	for _, token := range tokens {
		idx := strings.Index(lower, strings.ToLower(token))
		if idx >= 0 {
			score += 2
			if bestIdx == -1 || idx < bestIdx {
				bestIdx = idx
			}
		}
		if strings.Contains(lowerPath, strings.ToLower(token)) {
			score += 1
		}
	}
	if bestIdx == -1 {
		return score, ""
	}
	start := bestIdx - 220
	if start < 0 {
		start = 0
	}
	end := bestIdx + 520
	if end > len(content) {
		end = len(content)
	}
	return score, strings.TrimSpace(content[start:end])
}

func pairedPlannerPaths(path string) []string {
	out := []string{}
	if strings.HasSuffix(path, "_test.go") {
		out = append(out, strings.TrimSuffix(path, "_test.go")+".go")
	}
	if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
		out = append(out, strings.TrimSuffix(path, ".go")+"_test.go")
	}
	if strings.Contains(path, "/tests/") {
		out = append(out, strings.Replace(path, "/tests/", "/", 1))
	}
	return dedupeStrings(out)
}

func firstPlannerSnippet(content string) string {
	if len(content) > 700 {
		content = content[:700]
	}
	return strings.TrimSpace(content)
}

func (p AnthropicPlanner) Synthesize(ctx context.Context, req PlannerRequest) (PlannerResponse, error) {
	schemaMap, err := plannerResponseSchema()
	if err != nil {
		return PlannerResponse{}, err
	}
	prompt := plannerPrompt(req)
	msg, err := p.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:        p.model,
		MaxTokens:    1800,
		Messages:     []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt))},
		OutputFormat: anthropic.BetaJSONSchemaOutputFormat(schemaMap),
		Betas:        []anthropic.AnthropicBeta{"structured-outputs-2025-11-13"},
	})
	if err != nil {
		return PlannerResponse{}, err
	}
	for _, block := range msg.Content {
		switch v := block.AsAny().(type) {
		case anthropic.BetaToolUseBlock:
			var resp PlannerResponse
			if err := json.Unmarshal([]byte(v.JSON.Input.Raw()), &resp); err != nil {
				return PlannerResponse{}, err
			}
			return resp, nil
		case anthropic.BetaTextBlock:
			var resp PlannerResponse
			if err := json.Unmarshal([]byte(v.Text), &resp); err == nil {
				return resp, nil
			}
		}
	}
	return PlannerResponse{}, fmt.Errorf("planner returned no structured response blocks")
}

func plannerResponseSchema() (map[string]any, error) {
	reflector := jsonschema.Reflector{AllowAdditionalProperties: false, DoNotReference: true}
	schema := reflector.Reflect(&PlannerResponse{})
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func plannerPrompt(req PlannerRequest) string {
	var b strings.Builder
	b.WriteString("You are Airlock's repair planner. Generate between 1 and 5 bounded candidate mutations as strict JSON matching the provided schema. Do not explain outside JSON.\n")
	b.WriteString("Only use mutation kinds listed in allowedMutations. Prefer the smallest localized change that plausibly fixes the reproduced failure.\n")
	b.WriteString("Do not invent files. Use paths relative to the selected target path. Reuse exact oldText/oldLine/anchorText from snippets when possible.\n\n")
	b.WriteString("## Bug signal\n")
	b.WriteString(fmt.Sprintf("repoType: %s\n", req.Investigation.Profile.RepoType))
	b.WriteString(fmt.Sprintf("targetPath: %s\n", req.Investigation.Profile.TargetPath))
	if req.Input.IssueURL != "" {
		b.WriteString(fmt.Sprintf("issueUrl: %s\n", req.Input.IssueURL))
	}
	if req.Input.FailingCommand != "" {
		b.WriteString(fmt.Sprintf("failingCommand: %s\n", req.Input.FailingCommand))
	}
	if req.ValidationCommand != "" {
		b.WriteString(fmt.Sprintf("validationCommand: %s\n", req.ValidationCommand))
	}
	if req.Input.FailureText != "" {
		b.WriteString(fmt.Sprintf("failureText: %s\n", req.Input.FailureText))
	}
	if req.Input.Notes != "" {
		b.WriteString(fmt.Sprintf("notes: %s\n", req.Input.Notes))
	}
	if len(req.RankedMutationKinds) > 0 {
		b.WriteString("rankedMutationKinds:\n")
		for _, item := range req.RankedMutationKinds {
			b.WriteString(fmt.Sprintf("- %s (score=%d reasons=%s)\n", item.Kind, item.Score, strings.Join(item.Reasons, "; ")))
		}
	}
	b.WriteString(fmt.Sprintf("allowedMutations: %s\n", strings.Join(req.AllowedMutations, ", ")))
	b.WriteString("\n## Candidate file snippets\n")
	for _, file := range req.CandidateFiles {
		b.WriteString(fmt.Sprintf("### %s\n%s\n\n", file.Path, file.Snippet))
	}
	b.WriteString("Return bounded candidate attempts only. Each attempt must target one file and preserve reviewability.\n")
	return b.String()
}

func synthesizeWithPlanner(ctx context.Context, client PlannerClient, input PlanInput, report PlanReport, validationCmd string) ([]SynthesizedAttempt, string, error) {
	req := plannerRequestFor(input, report, validationCmd)
	resp, err := client.Synthesize(ctx, req)
	if err != nil {
		return nil, "", err
	}
	attempts, err := normalizePlannerAttempts(resp, report.Investigation.Profile, validationCmd)
	if err != nil {
		return nil, "", err
	}
	return attempts, resp.Summary, nil
}

func normalizePlannerAttempts(resp PlannerResponse, profile RepoProfile, validationCmd string) ([]SynthesizedAttempt, error) {
	out := []SynthesizedAttempt{}
	for _, item := range resp.Attempts {
		mutation, err := plannerMutationSpec(item)
		if err != nil {
			return nil, fmt.Errorf("planner attempt %q: %w", item.Name, err)
		}
		if err := validatePlannerPath(profile.TargetPath, item.Path); err != nil {
			return nil, fmt.Errorf("planner attempt %q: %w", item.Name, err)
		}
		attempt := AttemptFile{
			Attempt: AttemptSpec{
				Name:          utilSafePlannerName(item.Name, item.MutationKind),
				CommitMessage: fmt.Sprintf("attempt: %s", item.Name),
				Validation:    Phase{Command: validationCmd, Repeat: 1, Success: SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)}},
				Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: plannerMaxLOC(item.MutationKind), AllowedPaths: []string{item.Path}},
			},
			Mutation: mutation,
		}
		if errs := ValidateAttemptFile(AttemptFile{Repo: profile.TargetPath, ArtifactsDir: "/tmp/airlock-planner-validate", Attempt: attempt.Attempt, Mutation: attempt.Mutation}); len(errs) > 0 {
			return nil, fmt.Errorf("planner attempt %q invalid: %s", item.Name, strings.Join(errs, "; "))
		}
		out = append(out, SynthesizedAttempt{Name: item.Name, MutationKind: item.MutationKind, Confidence: item.Confidence, Rationale: item.Rationale, Attempt: attempt})
	}
	return out, nil
}

func plannerMutationSpec(item PlannerAttemptProposal) (MutationSpec, error) {
	switch item.MutationKind {
	case "search_replace":
		if item.Path == "" || item.OldText == "" {
			return MutationSpec{}, fmt.Errorf("search_replace requires path and oldText")
		}
		return MutationSpec{SearchReplace: &SearchReplaceMutation{Path: item.Path, OldText: item.OldText, NewText: item.NewText}}, nil
	case "replace_line":
		if item.Path == "" || item.OldLine == "" {
			return MutationSpec{}, fmt.Errorf("replace_line requires path and oldLine")
		}
		return MutationSpec{ReplaceLine: &ReplaceLineMutation{Path: item.Path, OldLine: item.OldLine, NewLine: item.NewLine}}, nil
	case "insert_after":
		if item.Path == "" || item.AnchorText == "" {
			return MutationSpec{}, fmt.Errorf("insert_after requires path and anchorText")
		}
		return MutationSpec{InsertAfter: &InsertAfterMutation{Path: item.Path, AnchorText: item.AnchorText, InsertText: item.InsertText}}, nil
	case "ensure_line":
		if item.Path == "" || item.Line == "" {
			return MutationSpec{}, fmt.Errorf("ensure_line requires path and line")
		}
		return MutationSpec{EnsureLine: &EnsureLineMutation{Path: item.Path, Line: item.Line}}, nil
	case "nil_guard":
		if item.Path == "" || item.AnchorText == "" || item.GuardLine == "" {
			return MutationSpec{}, fmt.Errorf("nil_guard requires path, anchorText, and guardLine")
		}
		return MutationSpec{NilGuard: &NilGuardMutation{Path: item.Path, AnchorText: item.AnchorText, GuardLine: item.GuardLine, InsertAfter: item.InsertAfter}}, nil
	case "error_return":
		if item.Path == "" || item.AnchorText == "" || item.ReturnLine == "" {
			return MutationSpec{}, fmt.Errorf("error_return requires path, anchorText, and returnLine")
		}
		return MutationSpec{ErrorReturn: &ErrorReturnMutation{Path: item.Path, AnchorText: item.AnchorText, ReturnLine: item.ReturnLine, InsertAfter: item.InsertAfter}}, nil
	default:
		return MutationSpec{}, fmt.Errorf("unsupported mutation kind %q", item.MutationKind)
	}
}

func validatePlannerPath(root, rel string) error {
	if rel == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") || rel == ".." {
		return fmt.Errorf("path must stay relative to target path")
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(full)
	if err != nil {
		return fmt.Errorf("path does not exist in target scope: %s", rel)
	}
	if info.IsDir() {
		return fmt.Errorf("path must reference a file: %s", rel)
	}
	return nil
}

func plannerMaxLOC(kind string) int {
	switch kind {
	case "nil_guard", "error_return", "insert_after":
		return 20
	default:
		return 30
	}
}

func utilSafePlannerName(name, fallback string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return fallback + "-attempt"
	}
	return name
}
