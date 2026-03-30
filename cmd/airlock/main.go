package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/research"
	"github.com/vmaliwal/airlock/internal/runner"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "check":
		runCheck()
	case "probe":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock probe <repo-path>")
			os.Exit(1)
		}
		runProbe(os.Args[2])
	case "investigate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock investigate <repo-path>")
			os.Exit(1)
		}
		runInvestigate(os.Args[2])
	case "plan":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock plan <repo-path|plan-input.json>")
			os.Exit(1)
		}
		runPlan(os.Args[2])
	case "intake-compile":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock intake-compile <repo-path|plan-input.json> [output.json]")
			os.Exit(1)
		}
		out := ""
		if len(os.Args) >= 4 {
			out = os.Args[3]
		}
		runIntakeCompile(os.Args[2], out)
	case "synthesize":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock synthesize <repo-path|plan-input.json> [output.json]")
			os.Exit(1)
		}
		out := ""
		if len(os.Args) >= 4 {
			out = os.Args[3]
		}
		runSynthesize(os.Args[2], out)
	case "eval-planner":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock eval-planner <cases.json> [output.json]")
			os.Exit(1)
		}
		out := ""
		if len(os.Args) >= 4 {
			out = os.Args[3]
		}
		runEvalPlanner(os.Args[2], out)
	case "fix":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock fix <github-issue-url>")
			os.Exit(1)
		}
		runFix(os.Args[2])
	case "metrics":
		path := ""
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		runMetrics(path)
	case "preflight":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock preflight <repo-path>")
			os.Exit(1)
		}
		runPreflight(os.Args[2])
	case "template":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock template <research|campaign|attempt|autofix>")
			os.Exit(1)
		}
		runTemplate(os.Args[2])
	case "attempt-run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock attempt-run <attempt.json>")
			os.Exit(1)
		}
		runAttempt(os.Args[2])
	case "autofix-run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock autofix-run <autofix.json>")
			os.Exit(1)
		}
		runAutofix(os.Args[2])
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock validate <contract.json>")
			os.Exit(1)
		}
		runValidate(os.Args[2])
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock run <contract.json>")
			os.Exit(1)
		}
		runContract(os.Args[2])
	case "research-validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock research-validate <research-contract.json>")
			os.Exit(1)
		}
		runResearchValidate(os.Args[2])
	case "research-run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock research-run <research-contract.json>")
			os.Exit(1)
		}
		runResearchRun(os.Args[2])
	case "campaign-validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock campaign-validate <campaign-plan.json|research-contract.json>")
			os.Exit(1)
		}
		runCampaignValidate(os.Args[2])
	case "campaign-run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: airlock campaign-run <campaign-plan.json|research-contract.json>")
			os.Exit(1)
		}
		runCampaignRun(os.Args[2])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: airlock <check|probe|investigate|plan|intake-compile|synthesize|eval-planner|fix|metrics|preflight|template|attempt-run|autofix-run|validate|run|research-validate|research-run|campaign-validate|campaign-run> [contract.json]")
}

func runCheck() {
	results := map[string][]string{}
	for _, kind := range []contract.BackendKind{contract.BackendLima, contract.BackendFirecracker} {
		b, err := runner.NewBackend(kind)
		if err != nil {
			results[string(kind)] = []string{err.Error()}
			continue
		}
		results[string(kind)] = b.CheckPrereqs()
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(data))
}

func runProbe(path string) {
	profile, err := research.DetectRepo(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	assessment, err := research.AssessRepo(profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(map[string]any{
		"profile":    profile,
		"assessment": assessment,
	}, "", "  ")
	fmt.Println(string(data))
}

func runInvestigate(path string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	report, err := research.InvestigateRepo(path, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(report))
}

func runPlan(arg string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	input, err := research.ResolvePlanInput(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := research.PlanFromInput(input, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(report))
}

func runIntakeCompile(arg, out string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	input, err := research.ResolvePlanInput(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rc, err := research.CompilePlanInputToRunContract(input, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if out == "" {
		fmt.Println(toJSON(rc))
		return
	}
	data, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]any{"output": out}))
}

func runSynthesize(arg, out string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	input, err := research.ResolvePlanInput(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := research.SynthesizeAutofixPlan(input, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if out == "" {
		fmt.Println(toJSON(report))
		return
	}
	if report.AutofixPlan == nil {
		fmt.Fprintln(os.Stderr, "no synthesized autofix plan available for this bug signal yet")
		os.Exit(1)
	}
	data, err := json.MarshalIndent(report.AutofixPlan, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]any{"output": out, "supported": report.Supported, "attempts": len(report.Attempts)}))
}

func runEvalPlanner(path, out string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	cases, err := research.LoadPlannerEvalCases(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	summary, err := research.RunPlannerEvalCases(cases, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if out == "" {
		fmt.Println(toJSON(summary))
		return
	}
	data, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]any{"output": out, "cases": summary.CaseCount}))
}

func runFix(issueURL string) {
	progress := func(stage, msg string, done bool, detail string) {
		status := "→"
		if done {
			status = "✓"
		}
		if detail != "" {
			fmt.Fprintf(os.Stderr, "%s %s... %s (%s)\n", status, stage, msg, detail)
			return
		}
		fmt.Fprintf(os.Stderr, "%s %s... %s\n", status, stage, msg)
	}
	start := time.Now()
	summary := research.RunSummary{
		RunID:          research.NewRunID("fix"),
		Timestamp:      start.UTC().Format(time.RFC3339),
		CustomerID:     research.CurrentCustomerID(),
		Entrypoint:     "fix",
		AirlockVersion: research.AirlockVersion(),
		IssueKey:       issueURL,
		RoundCount:     1,
	}
	appendSummary := func() {
		summary.DurationSeconds = int64(time.Since(start).Seconds())
		if err := research.AppendRunSummary(summary); err != nil {
			fmt.Fprintf(os.Stderr, "warning: append run summary: %v\n", err)
		}
	}
	fail := func(category string, err error) {
		summary.FailureCategory = category
		appendSummary()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	progress("Resolve issue", issueURL, false, "")
	issue, err := research.ResolveGitHubIssue(issueURL)
	if err != nil {
		fail("issue_resolution_failed", err)
	}
	summary.RepoKey = fmt.Sprintf("%s/%s", issue.Owner, issue.Repo)
	summary.IssueKey = fmt.Sprintf("%s/%s#%d", issue.Owner, issue.Repo, issue.Number)
	progress("Resolve issue", issue.Title, true, summary.IssueKey)
	progress("Clone repo", issue.CloneURL, false, "")
	repoPath, err := research.CloneIssueRepo(issue)
	if err != nil {
		fail("clone_failed", err)
	}
	progress("Clone repo", repoPath, true, "")
	input := research.BuildPlanInputFromIssue(issue, repoPath)
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
		summary.Backend = backend
		progress("Route to VM", string(kind), true, "")
	}
	if sha, err := research.GitHeadSHA(repoPath); err == nil {
		summary.RepoSHA = sha
	}
	result := research.FixResult{Issue: issue, RepoPath: repoPath, PlanInput: input}
	proof := research.ProofState{ReproStatus: research.ReproStatusNotReproduced, ValidationScope: "target_only", FixConfidence: "none"}
	if input.FailingCommand != "" {
		progress("Reproduce bug", input.FailingCommand, false, "readonly VM run")
		rc, err := research.CompilePlanInputToRunContract(input, backend, research.HostExecutionExceptionDeclared())
		if err != nil {
			fail("readonly_compile_failed", err)
		}
		compiled, err := research.CompileRunContract(rc)
		if err != nil {
			fail("readonly_compile_failed", err)
		}
		summaryPath, err := research.ExecuteCompiledContract(compiled)
		if err != nil {
			progress("Reproduce bug", "failed", true, err.Error())
			fail("readonly_execution_failed", err)
		}
		result.ReadonlySummaryPath = summaryPath
		if repro, ok := research.ReadSiblingArtifact(summaryPath, "reproduction-results.json"); ok {
			result.ReproductionResults = repro
			if reproStatus, ok := repro["repro_status"].(string); ok {
				summary.ReproStatus = reproStatus
			}
			progress("Reproduce bug", "completed", true, fmt.Sprintf("repro_status=%v", repro["repro_status"]))
		} else {
			progress("Reproduce bug", "completed", true, summaryPath)
		}
		if proofState, ok := research.ReadSiblingArtifact(summaryPath, "proof-state.json"); ok {
			if v, ok := proofState["repro_status"].(string); ok {
				proof.ReproStatus = v
				summary.ReproStatus = v
			}
			if v, ok := proofState["validation_scope"].(string); ok {
				proof.ValidationScope = v
			}
			if v, ok := proofState["fix_confidence"].(string); ok {
				proof.FixConfidence = v
			}
		}
	} else {
		progress("Reproduce bug", "skipped", true, "no failing command inferred from issue")
		summary.ReproStatus = research.ReproStatusNotReproduced
		proof.ReproStatus = research.ReproStatusNotReproduced
	}
	maxFixRounds := 3
	var latestSynth research.SynthesisReport
	loop, loopErr := research.RunAutofixLoop(research.AutofixLoopPolicy{MaxRounds: maxFixRounds}, func(round int, previous *research.AutofixSummary) (*research.AutofixPlan, error) {
		roundInput := research.BuildNextRoundPlanInput(input, previous)
		progress("Generate candidates", fmt.Sprintf("round %d synthesizing", round), false, "")
		synth, err := research.SynthesizeAutofixPlan(roundInput, backend, research.HostExecutionExceptionDeclared())
		if err != nil {
			return nil, err
		}
		latestSynth = synth
		result.Synthesis = synth
		progress("Generate candidates", fmt.Sprintf("round %d found %d attempts", round, len(synth.Attempts)), true, synth.Summary)
		if synth.AutofixPlan == nil {
			return &research.AutofixPlan{Objective: synth.Input.FailureText, Repo: input.RepoPath, ArtifactsDir: filepath.Join(os.TempDir(), "airlock-empty-fix-loop"), Attempts: nil}, nil
		}
		plan := *synth.AutofixPlan
		plan.ArtifactsDir = filepath.Join(plan.ArtifactsDir, "fix-loop")
		return &plan, nil
	}, func(round int, plan research.AutofixPlan) (string, error) {
		progress("Attempt execution", fmt.Sprintf("round %d running %d attempts", round, len(plan.Attempts)), false, "")
		policy, handled := decideExecutionPolicy(plan.Repo)
		if handled && policy.Preflight.Route == "vm" {
			compiled, err := research.CompileAutofixPlanToVMContract(plan, policy.BackendKind)
			if err != nil {
				return "", err
			}
			summaryPath, err := executeBaseContract(compiled)
			if err != nil {
				return "", err
			}
			if round == maxFixRounds || result.AutofixContractSummary == "" {
				if autofixResult, ok := research.ReadSiblingArtifact(summaryPath, "autofix-result.json"); ok {
					result.AutofixResult = autofixResult
				}
				if proofState, ok := research.ReadSiblingArtifact(summaryPath, "proof-state.json"); ok {
					if v, ok := proofState["repro_status"].(string); ok {
						proof.ReproStatus = v
						summary.ReproStatus = v
					}
					if v, ok := proofState["validation_scope"].(string); ok {
						proof.ValidationScope = v
					}
					if v, ok := proofState["fix_confidence"].(string); ok {
						proof.FixConfidence = v
					}
				}
			}
			return summaryPath, nil
		}
		summaryPath, err := research.RunAutofixPlan(plan)
		if err == nil {
			proof.ValidationScope = "target_only"
			proof.FixConfidence = "medium"
		}
		return summaryPath, err
	})
	result.FixLoop = loop
	result.Synthesis = latestSynth
	summary.AttemptCount = len(latestSynth.Attempts)
	if len(loop.Rounds) > 0 {
		summary.RoundCount = len(loop.Rounds)
	}
	if loop.FinalSummaryPath != "" {
		result.AutofixContractSummary = loop.FinalSummaryPath
	}
	if loopErr == nil && result.AutofixContractSummary != "" {
		progress("Attempt execution", "completed", true, result.AutofixContractSummary)
		proof.ValidationScope = "target_only"
		proof.FixConfidence = "medium"
	} else if loopErr != nil {
		if len(loop.Rounds) > 0 {
			last := loop.Rounds[len(loop.Rounds)-1]
			if last.StopReason == "no_new_attempts" {
				summary.FailureCategory = "no_new_attempts"
			} else {
				summary.FailureCategory = "no_candidate_fix"
			}
		} else {
			summary.FailureCategory = "no_candidate_fix"
		}
		if result.AutofixContractSummary == "" {
			summary.ValidationScope = proof.ValidationScope
			summary.FixConfidence = proof.FixConfidence
			if packetPath, packetErr := research.WriteReviewPacket(result, summary, proof); packetErr == nil {
				result.ReviewPacketPath = packetPath
				if draftPath, draftErr := research.WriteDraftPRArtifact(result, summary, proof); draftErr == nil {
					result.DraftPRPath = draftPath
				}
			}
			appendSummary()
			fmt.Println(toJSON(result))
			return
		}
	}
	if result.AutofixContractSummary != "" && proof.ValidationScope == "reproduction_only" {
		proof.ValidationScope = "target_only"
		if proof.ReproStatus == research.ReproStatusReproduced {
			proof.FixConfidence = "medium"
		} else {
			proof.FixConfidence = "low"
		}
	}
	decision := research.DecideAdvancement(proof, result.AutofixContractSummary != "", true, false)
	summary.Advance = decision.ShouldAdvance
	summary.CredibleAdvancement = decision.CredibleAdvancement
	summary.VerifiedIssueResolution = decision.VerifiedIssueResolution
	summary.ValidationScope = proof.ValidationScope
	summary.FixConfidence = proof.FixConfidence
	if summary.ReproStatus == "" {
		summary.ReproStatus = proof.ReproStatus
	}
	if decision.FailureCategory != "" {
		summary.FailureCategory = decision.FailureCategory
	}
	if result.AutofixContractSummary != "" {
		if data, err := os.ReadFile(result.AutofixContractSummary); err == nil {
			var direct map[string]any
			if json.Unmarshal(data, &direct) == nil {
				if v, ok := direct["winningAttempt"].(string); ok {
					summary.WinningAttempt = v
				}
			}
		}
		if summary.WinningAttempt == "" {
			if autofixSummary, ok := research.ReadSiblingArtifact(result.AutofixContractSummary, "autofix-summary.json"); ok {
				if v, ok := autofixSummary["winningAttempt"].(string); ok {
					summary.WinningAttempt = v
				}
			}
		}
	}
	if packetPath, packetErr := research.WriteReviewPacket(result, summary, proof); packetErr == nil {
		result.ReviewPacketPath = packetPath
		progress("Review packet", "written", true, packetPath)
		if draftPath, draftErr := research.WriteDraftPRArtifact(result, summary, proof); draftErr == nil {
			result.DraftPRPath = draftPath
			progress("Draft PR", "written", true, draftPath)
		}
	}
	appendSummary()
	progress("Done", "finished", true, time.Since(start).Round(time.Second).String())
	fmt.Println(toJSON(result))
}

func runMetrics(path string) {
	if path == "" {
		path = research.DefaultRunLedgerPath()
	}
	items, err := research.LoadRunSummaries(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	summary := research.SummarizeRunMetrics(items)
	summary.LedgerPath = path
	fmt.Println(toJSON(summary))
}

func runPreflight(path string) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	decision, err := research.PreflightRepo(path, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(decision))
}

func runTemplate(kind string) {
	switch kind {
	case "research":
		fmt.Print(research.ResearchContractTemplate)
	case "campaign":
		fmt.Print(research.CampaignPlanTemplate)
	case "attempt":
		fmt.Print(research.AttemptTemplate)
	case "autofix":
		fmt.Print(research.AutofixTemplate)
	default:
		fmt.Fprintln(os.Stderr, "usage: airlock template <research|campaign|attempt|autofix>")
		os.Exit(1)
	}
}

func maybeRouteAttemptToVM(cfg research.AttemptFile) bool {
	policy, handled := decideExecutionPolicy(cfg.Repo)
	if !handled {
		return false
	}
	if policy.Preflight.Route == "vm" {
		compiled, cerr := research.CompileAttemptFileToVMContract(cfg, policy.BackendKind)
		if cerr != nil {
			fmt.Fprintln(os.Stderr, cerr)
			os.Exit(1)
		}
		summaryPath, rerr := executeBaseContract(compiled)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, rerr)
			os.Exit(1)
		}
		fmt.Println(toJSON(map[string]any{"summaryPath": summaryPath, "routedToVM": true, "backend": policy.BackendKind, "hostExecutionExceptionDeclared": research.HostExecutionExceptionDeclared()}))
		return true
	}
	return false
}

func maybeRouteAutofixToVM(cfg research.AutofixPlan) bool {
	policy, handled := decideExecutionPolicy(cfg.Repo)
	if !handled {
		return false
	}
	if policy.Preflight.Route == "vm" {
		compiled, cerr := research.CompileAutofixPlanToVMContract(cfg, policy.BackendKind)
		if cerr != nil {
			fmt.Fprintln(os.Stderr, cerr)
			os.Exit(1)
		}
		summaryPath, rerr := executeBaseContract(compiled)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, rerr)
			os.Exit(1)
		}
		fmt.Println(toJSON(map[string]any{"summaryPath": summaryPath, "routedToVM": true, "backend": policy.BackendKind, "hostExecutionExceptionDeclared": research.HostExecutionExceptionDeclared()}))
		return true
	}
	return false
}

func decideExecutionPolicy(repo string) (research.ExecutionPolicyDecision, bool) {
	backend := ""
	if kind, err := selectAutoVMBackend(); err == nil {
		backend = string(kind)
	}
	policy, err := research.DecideExecutionPolicy(repo, backend, research.HostExecutionExceptionDeclared())
	if err != nil {
		return research.ExecutionPolicyDecision{}, false
	}
	if policy.Preflight.Route == "vm" {
		if policy.BackendKind == "" {
			fmt.Fprintln(os.Stderr, "host execution blocked by policy and no VM backend is ready")
			fmt.Fprintf(os.Stderr, "declare %s=1 only for an explicit host exception\n", research.HostExecutionExceptionEnv)
			os.Exit(1)
		}
	}
	return policy, true
}

func selectAutoVMBackend() (contract.BackendKind, error) {
	if runtime.GOOS == "darwin" {
		if _, err := runner.NewBackend(contract.BackendLima); err == nil {
			return contract.BackendLima, nil
		}
	}
	if _, err := runner.NewBackend(contract.BackendFirecracker); err == nil {
		return contract.BackendFirecracker, nil
	}
	return "", fmt.Errorf("no VM backend available")
}

func runAttempt(path string) {
	cfg, err := research.LoadAttemptFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateAttemptFile(cfg); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	if maybeRouteAttemptToVM(cfg) {
		return
	}
	outcome, err := research.RunAttemptFile(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(outcome))
}

func runAutofix(path string) {
	cfg, err := research.LoadAutofixPlan(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateAutofixPlan(cfg); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	if maybeRouteAutofixToVM(cfg) {
		return
	}
	summary, err := research.RunAutofixPlan(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]string{"summaryPath": summary}))
}

func runValidate(path string) {
	c, err := contract.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := contract.Validate(c); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	fmt.Println(toJSON(c))
}

func runContract(path string) {
	c, err := contract.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := contract.Validate(c); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	summaryPath, err := executeBaseContract(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
}

func runResearchValidate(path string) {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(compiled))
}

func runResearchRun(path string) {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	summaryPath, err := executeBaseContract(compiled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
}

func runCampaignValidate(path string) {
	if filepath.Ext(path) == ".json" && stringsContains(path, "campaign") {
		plan, err := research.LoadCampaignPlan(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if errs := research.ValidateCampaignPlan(plan); len(errs) > 0 {
			for _, msg := range errs {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(1)
		}
		fmt.Println(toJSON(plan))
		return
	}
	rc := mustLoadCampaignContract(path)
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(compiled))
}

func runCampaignRun(path string) {
	if filepath.Ext(path) == ".json" && stringsContains(path, "campaign") {
		plan, err := research.LoadCampaignPlan(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if errs := research.ValidateCampaignPlan(plan); len(errs) > 0 {
			for _, msg := range errs {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(1)
		}
		summary, err := research.RunCampaignPlan(path, plan)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(toJSON(map[string]string{"summaryPath": summary}))
		return
	}
	rc := mustLoadCampaignContract(path)
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
	summaryPath, err := executeBaseContract(compiled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
	fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
}

func mustLoadCampaignContract(path string) research.RunContract {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		for _, msg := range errs {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}
	if rc.Campaign == nil {
		fmt.Fprintln(os.Stderr, "campaign section is required for single-contract campaign mode")
		os.Exit(1)
	}
	return rc
}

func stringsContains(s, want string) bool {
	return len(s) >= len(want) && filepath.Base(s) != "" && (s == want || filepath.Ext(s) == filepath.Ext(want) && filepath.Base(s) != "") && (func() bool { return true })() && (len(want) == 0 || len(s) > 0) && (func() bool { return stringsIndex(s, want) >= 0 })()
}

func stringsIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

func executeBaseContract(c contract.Contract) (string, error) {
	return research.ExecuteCompiledContract(c)
}
