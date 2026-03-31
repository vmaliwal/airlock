package research

import (
	"fmt"
	"path/filepath"
	"strings"

	base "github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/util"
)

func CompilePlanInputToRunContract(input PlanInput, vmBackend string, allowHostExecution bool) (RunContract, error) {
	report, err := PlanFromInput(input, vmBackend, allowHostExecution)
	if err != nil {
		return RunContract{}, err
	}
	profile := report.Investigation.Profile
	cloneURL, err := GitRemoteOrigin(profile.TargetPath)
	if err != nil {
		return RunContract{}, fmt.Errorf("detect origin remote: %w", err)
	}
	cloneURL = NormalizeCloneURL(cloneURL)
	headSHA, err := GitHeadSHA(profile.RepoRoot)
	if err != nil {
		return RunContract{}, fmt.Errorf("detect head sha: %w", err)
	}
	relSubdir, err := filepath.Rel(profile.RepoRoot, profile.ScopeRoot)
	if err != nil {
		return RunContract{}, fmt.Errorf("resolve subdir: %w", err)
	}
	if relSubdir == "." {
		relSubdir = ""
	}
	backendKind := base.BackendLima
	if vmBackend == string(base.BackendFirecracker) {
		backendKind = base.BackendFirecracker
	}
	objective := compiledObjective(input)
	cmd := compiledTargetCommand(input, report)
	cmd = applyRuntimeBootstrapPolicy(profile, cmd)
	name := compiledNamePrefix(input, profile)
	artifactsDir := filepath.ToSlash(filepath.Join("/tmp", "airlock-"+name))

	var rc RunContract
	rc.Objective = objective
	rc.Mode = "read_only"
	rc.HostExecutionException = allowHostExecution
	rc.Setup = append(issueReproSetupSteps(input), defaultBootstrapSetup(profile)...)
	rc.Reproduction = Phase{Command: cmd, Repeat: 1, Success: SuccessRule{MinFailures: pintCompiled(1)}}
	rc.Validation = ValidationSpec{
		TargetCommand: cmd,
		Repeat:        1,
		Success:       SuccessRule{ExitCode: pintCompiled(0), MinPassRate: pfloatCompiled(1.0), MaxFailures: pintCompiled(0)},
	}
	rc.Stuck = StuckPolicy{MaxSameFailureFingerprint: 2, MaxAttemptsWithoutImprovement: 2}
	rc.Airlock.Backend.Kind = backendKind
	rc.Airlock.Sandbox.NamePrefix = name
	rc.Airlock.Sandbox.ArtifactsDir = artifactsDir
	rc.Airlock.Sandbox.CPU = 4
	rc.Airlock.Sandbox.MemoryGiB = 8
	rc.Airlock.Sandbox.DiskGiB = 20
	rc.Airlock.Sandbox.TTLMinutes = 60
	rc.Airlock.Repo.CloneURL = cloneURL
	rc.Airlock.Repo.Ref = headSHA
	rc.Airlock.Repo.Subdir = filepath.ToSlash(relSubdir)
	rc.Airlock.Security.BootstrapNetwork = base.NetworkAllowlist
	rc.Airlock.Security.BootstrapAllowHosts = []string{"archive.ubuntu.com", "security.ubuntu.com", "ports.ubuntu.com"}
	rc.Airlock.Security.BootstrapAptPackages = bootstrapAptPackagesFor(profile.RepoType)
	rc.Airlock.Security.Network = base.NetworkAllowlist
	rc.Airlock.Security.AllowHosts = networkAllowHostsFor(profile.RepoType)
	rc.Airlock.Security.AllowedEnv = compiledAllowedEnv(cloneURL)
	rc.Airlock.Security.ExportPaths = []string{"/airlock/artifacts"}
	rc.Airlock.Security.IncludePatch = true
	rc.Airlock.Steps = []base.Step{{Name: "placeholder", Run: "true"}}
	return rc, nil
}

func CompiledResearchContractPath(input PlanInput) string {
	name := util.SafeName(compiledObjective(input))
	return filepath.ToSlash(filepath.Join(".", name+"-research.json"))
}

func compiledObjective(input PlanInput) string {
	parts := []string{}
	if input.IssueURL != "" {
		parts = append(parts, input.IssueURL)
	}
	if input.FailureText != "" {
		parts = append(parts, input.FailureText)
	} else if input.Notes != "" {
		parts = append(parts, input.Notes)
	} else if input.FailingCommand != "" {
		parts = append(parts, input.FailingCommand)
	}
	if len(parts) == 0 {
		parts = append(parts, input.RepoPath)
	}
	return strings.Join(parts, " — ")
}

func compiledTargetCommand(input PlanInput, report PlanReport) string {
	if input.FailingCommand != "" {
		return input.FailingCommand
	}
	if len(report.Investigation.CandidateReproduction) > 0 {
		return report.Investigation.CandidateReproduction[0]
	}
	if len(report.Investigation.CandidateValidation) > 0 {
		return report.Investigation.CandidateValidation[0]
	}
	return "true"
}

func applyRuntimeBootstrapPolicy(profile RepoProfile, cmd string) string {
	switch profile.RepoType {
	case "python":
		if strings.HasPrefix(cmd, ".venv/bin/python ") {
			return cmd
		}
		if strings.HasPrefix(cmd, "python3 -m ") {
			return strings.Replace(cmd, "python3 -m ", ".venv/bin/python -m ", 1)
		}
		if strings.HasPrefix(cmd, "python -m ") {
			return strings.Replace(cmd, "python -m ", ".venv/bin/python -m ", 1)
		}
		if cmd == "pytest" || strings.HasPrefix(cmd, "pytest ") {
			return strings.Replace(cmd, "pytest", ".venv/bin/python -m pytest", 1)
		}
		return cmd
	case "go":
		return applyBootstrapPrefix(cmd, goToolchainBootstrapCommand(profile))
	case "node":
		// No command rewriting needed for node — install is handled by the
		// dedicated bootstrap setup step. Commands like npm test / pnpm test /
		// node repro.js run as-is after the bootstrap step succeeds.
		return cmd
	default:
		return cmd
	}
}

func defaultBootstrapSetup(profile RepoProfile) []SetupStep {
	switch profile.RepoType {
	case "python":
		return pythonBootstrapSetup(profile)
	case "node":
		return nodeBootstrapSetup(profile)
	default:
		return nil
	}
}

func pythonBootstrapSetup(profile RepoProfile) []SetupStep {
	commands := []string{"python3 -m venv .venv", ".venv/bin/python -m pip install -q --upgrade pip"}
	switch {
	case contains(profile.DetectedFiles, "requirements.txt"):
		commands = append(commands, ".venv/bin/python -m pip install -q -r requirements.txt")
	case contains(profile.DetectedFiles, "pyproject.toml"):
		commands = append(commands, ".venv/bin/python -m pip install -q -e .")
	}
	return []SetupStep{{
		Name:    "bootstrap python venv",
		Command: strings.Join(commands, " && "),
	}}
}

func nodeBootstrapSetup(profile RepoProfile) []SetupStep {
	cmd := nodeInstallCommand(profile.DetectedFiles)
	if cmd == "" {
		return nil
	}
	return []SetupStep{{
		Name:    "bootstrap node dependencies",
		Command: cmd,
	}}
}

func nodeInstallCommand(detectedFiles []string) string {
	switch {
	case contains(detectedFiles, "pnpm-lock.yaml"):
		return "npm install -g pnpm --silent && pnpm install --frozen-lockfile"
	case contains(detectedFiles, "yarn.lock"):
		return "yarn install --frozen-lockfile"
	case contains(detectedFiles, "package-lock.json"):
		return "npm ci --silent"
	default:
		// no lockfile — use npm install but don't freeze
		return "npm install --silent"
	}
}

func compiledNamePrefix(input PlanInput, profile RepoProfile) string {
	parts := []string{}
	if input.IssueURL != "" {
		parts = append(parts, util.SafeName(input.IssueURL))
	} else {
		parts = append(parts, util.SafeName(filepath.Base(profile.ScopeRoot)))
	}
	if input.FailureText != "" {
		parts = append(parts, util.SafeName(input.FailureText))
	}
	name := strings.Join(parts, "-")
	name = util.SafeName(name)
	if name == "" {
		return "research-intake"
	}
	return name
}

func bootstrapAptPackagesFor(repoType string) []string {
	basePkgs := []string{"git", "ca-certificates"}
	switch repoType {
	case "python":
		return append(basePkgs, "python3", "python3-pip", "python3-venv")
	case "go":
		return append(basePkgs, "curl")
	case "node":
		return append(basePkgs, "nodejs", "npm")
	default:
		return basePkgs
	}
}

func networkAllowHostsFor(repoType string) []string {
	switch repoType {
	case "python":
		return []string{"pypi.org", "files.pythonhosted.org"}
	case "go":
		return []string{"github.com", "go.dev", "dl.google.com", "proxy.golang.org", "sum.golang.org"}
	case "node":
		// npm, pnpm, and yarn all resolve from registry.npmjs.org by default.
		// registry.yarnpkg.com is an alias; include it for older yarn configs.
		return []string{"registry.npmjs.org", "registry.yarnpkg.com"}
	default:
		return []string{"github.com"}
	}
}

func compiledAllowedEnv(cloneURL string) []string {
	cloneURL = NormalizeCloneURL(cloneURL)
	if strings.HasPrefix(cloneURL, "https://github.com/") || strings.HasPrefix(cloneURL, "git@github.com:") {
		return []string{"GITHUB_TOKEN"}
	}
	return []string{}
}

func pintCompiled(v int) *int           { return &v }
func pfloatCompiled(v float64) *float64 { return &v }
