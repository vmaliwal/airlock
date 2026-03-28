package research

type SuccessRule struct {
	ExitCode          *int     `json:"exit_code,omitempty"`
	StdoutContains    []string `json:"stdout_contains,omitempty"`
	StderrNotContains []string `json:"stderr_not_contains,omitempty"`
	MinPassRate       *float64 `json:"min_pass_rate,omitempty"`
	MaxDurationMs     *int64   `json:"max_duration_ms,omitempty"`
	MinFailures       *int     `json:"min_failures,omitempty"`
	MaxFailures       *int     `json:"max_failures,omitempty"`
}

type EvaluationResult struct {
	Passed      bool     `json:"passed"`
	PassRate    float64  `json:"passRate,omitempty"`
	PassCount   int      `json:"passCount,omitempty"`
	FailCount   int      `json:"failCount,omitempty"`
	Observed    []string `json:"observed"`
	FailedRules []string `json:"failedRules"`
}

type CommandResult struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"durationMs"`
}

type RepoProfile struct {
	RepoPath          string   `json:"repoPath"`
	RepoRoot          string   `json:"repoRoot"`
	TargetPath        string   `json:"targetPath"`
	RepoType          string   `json:"repoType"`
	DetectedFiles     []string `json:"detectedFiles"`
	DiscoveredTargets []string `json:"discoveredTargets,omitempty"`
	BaselineCommands  []string `json:"baselineCommands"`
}

type RepoAssessment struct {
	Runnable             bool     `json:"runnable"`
	HostRunnable         bool     `json:"hostRunnable"`
	VMRunnable           bool     `json:"vmRunnable"`
	RecommendedExecution string   `json:"recommendedExecution"`
	Status               string   `json:"status"`
	PossibleModes        []string `json:"possibleModes"`
	Blockers             []string `json:"blockers"`
	Evidence             []string `json:"evidence"`
}

type FailureFingerprint struct {
	Kind      string   `json:"kind"`
	Signature string   `json:"signature"`
	Count     int      `json:"count"`
	Samples   []string `json:"samples"`
}
