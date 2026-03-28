package contract

type BackendKind string

type NetworkMode string

const (
	BackendLima        BackendKind = "lima"
	BackendFirecracker BackendKind = "firecracker"

	NetworkDeny      NetworkMode = "deny"
	NetworkAllowlist NetworkMode = "allowlist"
)

type FirecrackerHostConfig struct {
	Mode            string `json:"mode"`
	Host            string `json:"host,omitempty"`
	User            string `json:"user,omitempty"`
	Port            int    `json:"port,omitempty"`
	RemoteWorkDir   string `json:"remoteWorkDir,omitempty"`
	SSHIdentityFile string `json:"sshIdentityFile,omitempty"`
}

type Contract struct {
	Backend struct {
		Kind            BackendKind            `json:"kind"`
		FirecrackerHost *FirecrackerHostConfig `json:"firecrackerHost,omitempty"`
	} `json:"backend"`
	Sandbox struct {
		NamePrefix   string `json:"namePrefix"`
		ArtifactsDir string `json:"artifactsDir"`
		CPU          int    `json:"cpu"`
		MemoryGiB    int    `json:"memoryGiB"`
		DiskGiB      int    `json:"diskGiB"`
		TTLMinutes   int    `json:"ttlMinutes"`
	} `json:"sandbox"`
	Repo struct {
		CloneURL string `json:"cloneUrl"`
		Ref      string `json:"ref,omitempty"`
		Subdir   string `json:"subdir,omitempty"`
	} `json:"repo"`
	Security struct {
		BootstrapNetwork     NetworkMode `json:"bootstrapNetwork,omitempty"`
		BootstrapAllowHosts  []string    `json:"bootstrapAllowHosts,omitempty"`
		BootstrapAptPackages []string    `json:"bootstrapAptPackages,omitempty"`
		Network              NetworkMode `json:"network"`
		AllowHosts           []string    `json:"allowHosts"`
		AllowedEnv           []string    `json:"allowedEnv"`
		ExportPaths          []string    `json:"exportPaths"`
		IncludePatch         bool        `json:"includePatch,omitempty"`
	} `json:"security"`
	Steps []Step `json:"steps"`
}

type Step struct {
	Name           string `json:"name"`
	Run            string `json:"run"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
	AllowFailure   bool   `json:"allowFailure,omitempty"`
}

type StepResult struct {
	Name           string `json:"name"`
	Command        string `json:"command"`
	ExitCode       int    `json:"exitCode"`
	StdoutPath     string `json:"stdoutPath"`
	StderrPath     string `json:"stderrPath"`
	StartedAt      string `json:"startedAt"`
	FinishedAt     string `json:"finishedAt"`
	DurationMs     int64  `json:"durationMs"`
	AllowedFailure bool   `json:"allowedFailure"`
}

type RunSummary struct {
	Backend     BackendKind `json:"backend"`
	SandboxName string      `json:"sandboxName"`
	Repo        struct {
		CloneURL string `json:"cloneUrl"`
		Ref      string `json:"ref,omitempty"`
		Subdir   string `json:"subdir,omitempty"`
	} `json:"repo"`
	StartedAt        string       `json:"startedAt"`
	FinishedAt       string       `json:"finishedAt"`
	Success          bool         `json:"success"`
	Steps            []StepResult `json:"steps"`
	PatchPath        string       `json:"patchPath,omitempty"`
	GuestArtifactDir string       `json:"guestArtifactDir"`
}
