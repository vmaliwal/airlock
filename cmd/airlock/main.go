package main

import (
	"encoding/json"
	"fmt"
	"os"
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
			fmt.Fprintln(os.Stderr, "usage: airlock plan <repo-path>")
			os.Exit(1)
		}
		runPlan(os.Args[2])
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
	fmt.Println("usage: airlock <check|probe|investigate|plan|preflight|template|attempt-run|autofix-run|validate|run|research-validate|research-run|campaign-validate|campaign-run> [contract.json]")
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
		return policy, true
	}
	if policy.Preflight.Route == "stop" {
		fmt.Fprintln(os.Stderr, policy.Preflight.Reason)
		os.Exit(1)
	}
	if !research.HostExecutionExceptionDeclared() {
		fmt.Fprintf(os.Stderr, "host execution blocked by policy; declare %s=1 only for an explicit host exception\n", research.HostExecutionExceptionEnv)
		os.Exit(1)
	}
	return policy, false
}

func runAttempt(path string) {
	cfg, err := research.LoadAttemptFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateAttemptFile(cfg); len(errs) > 0 {
		fmt.Println(toJSON(errs))
		os.Exit(1)
	}
	if handled := maybeRouteAttemptToVM(cfg); handled {
		return
	}
	outcome, err := research.RunAttemptFile(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(toJSON(outcome))
	if !outcome.Success {
		os.Exit(1)
	}
}

func runAutofix(path string) {
	cfg, err := research.LoadAutofixPlan(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateAutofixPlan(cfg); len(errs) > 0 {
		fmt.Println(toJSON(errs))
		os.Exit(1)
	}
	if handled := maybeRouteAutofixToVM(cfg); handled {
		return
	}
	summaryPath, err := research.RunAutofixPlan(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
		os.Exit(1)
	}
	fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
}

func runValidate(path string) {
	c, err := contract.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	errs := contract.Validate(c)
	if len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		fmt.Println(string(data))
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	fmt.Println(string(data))
}

func runContract(path string) {
	c, err := contract.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := contract.Validate(c); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		fmt.Println(string(data))
		os.Exit(1)
	}
	runBaseContract(c)
}

func runResearchValidate(path string) {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		fmt.Println(string(data))
		os.Exit(1)
	}
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(compiled, "", "  ")
	fmt.Println(string(data))
}

func runResearchRun(path string) {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		fmt.Println(string(data))
		os.Exit(1)
	}
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runBaseContract(compiled)
}

func runCampaignValidate(path string) {
	if plan, ok := maybeLoadCampaignPlan(path); ok {
		if errs := research.ValidateCampaignPlan(plan); len(errs) > 0 {
			data, _ := json.MarshalIndent(errs, "", "  ")
			fmt.Println(string(data))
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(data))
		return
	}
	rc := mustLoadCampaignContract(path)
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(compiled, "", "  ")
	fmt.Println(string(data))
}

func runCampaignRun(path string) {
	if plan, ok := maybeLoadCampaignPlan(path); ok {
		if errs := research.ValidateCampaignPlan(plan); len(errs) > 0 {
			fmt.Println(toJSON(errs))
			os.Exit(1)
		}
		summaryPath, err := research.RunCampaignPlan(path, plan)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
			os.Exit(1)
		}
		fmt.Println(toJSON(map[string]string{"summaryPath": summaryPath}))
		return
	}
	rc := mustLoadCampaignContract(path)
	compiled, err := research.CompileRunContract(rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runBaseContract(compiled)
}

func maybeLoadCampaignPlan(path string) (research.CampaignPlan, bool) {
	plan, err := research.LoadCampaignPlan(path)
	if err != nil {
		return research.CampaignPlan{}, false
	}
	if len(plan.Entries) == 0 {
		return research.CampaignPlan{}, false
	}
	return plan, true
}

func mustLoadCampaignContract(path string) research.RunContract {
	rc, err := research.LoadRunContract(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if errs := research.ValidateRunContract(rc); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		fmt.Println(string(data))
		os.Exit(1)
	}
	if rc.Campaign == nil {
		fmt.Fprintln(os.Stderr, "campaign contract requires top-level campaign section")
		os.Exit(1)
	}
	return rc
}

func toJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

func runBaseContract(c contract.Contract) {
	summaryPath, err := executeBaseContract(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(map[string]string{"summaryPath": summaryPath}, "", "  ")
	fmt.Println(string(data))
}

func executeBaseContract(c contract.Contract) (string, error) {
	b, err := runner.NewBackend(c.Backend.Kind)
	if err != nil {
		return "", err
	}
	if errs := b.CheckPrereqs(); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		return "", fmt.Errorf(string(data))
	}
	result, err := b.Run(c)
	if err != nil {
		return "", err
	}
	return result.SummaryPath, nil
}

func selectAutoVMBackend() (contract.BackendKind, error) {
	candidates := []contract.BackendKind{contract.BackendLima, contract.BackendFirecracker}
	if runtime.GOOS == "linux" {
		candidates = []contract.BackendKind{contract.BackendFirecracker, contract.BackendLima}
	}
	for _, kind := range candidates {
		b, err := runner.NewBackend(kind)
		if err != nil {
			continue
		}
		if errs := b.CheckPrereqs(); len(errs) == 0 {
			return kind, nil
		}
	}
	return "", fmt.Errorf("repo is host-toolchain-blocked and should run in a VM, but no VM backend is ready; run 'airlock check'")
}
