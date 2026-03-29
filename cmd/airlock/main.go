package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	fmt.Println("usage: airlock <check|probe|investigate|plan|intake-compile|synthesize|preflight|template|attempt-run|autofix-run|validate|run|research-validate|research-run|campaign-validate|campaign-run> [contract.json]")
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
