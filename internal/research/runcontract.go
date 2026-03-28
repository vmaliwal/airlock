package research

import (
	"encoding/json"
	"os"

	base "github.com/vmaliwal/airlock/internal/contract"
)

type SetupStep struct {
	Name          string `json:"name"`
	Command       string `json:"command"`
	CommitMessage string `json:"commit_message,omitempty"`
}

type PatchStep struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type SafetyBudget struct {
	MaxFilesChanged int      `json:"max_files_changed,omitempty"`
	MaxLocChanged   int      `json:"max_loc_changed,omitempty"`
	AllowedPaths    []string `json:"allowed_paths,omitempty"`
	ForbiddenPaths  []string `json:"forbidden_paths,omitempty"`
}

type StuckPolicy struct {
	MaxSameFailureFingerprint     int `json:"max_same_failure_fingerprint,omitempty"`
	MaxAttemptsWithoutImprovement int `json:"max_attempts_without_improvement,omitempty"`
}

type CampaignSuccess struct {
	MaxTotalFailures             *int     `json:"max_total_failures,omitempty"`
	MustRemoveFingerprints       []string `json:"must_remove_fingerprints,omitempty"`
	MustNotIntroduceFingerprints bool     `json:"must_not_introduce_fingerprints,omitempty"`
}

type CampaignSpec struct {
	InventoryCommand string          `json:"inventory_command"`
	Success          CampaignSuccess `json:"success"`
}

type BroaderValidation struct {
	Name    string      `json:"name"`
	Command string      `json:"command"`
	Success SuccessRule `json:"success,omitempty"`
}

type ValidationSpec struct {
	TargetCommand   string              `json:"target_command"`
	Repeat          int                 `json:"repeat"`
	Success         SuccessRule         `json:"success"`
	NeighborCommand string              `json:"neighbor_command,omitempty"`
	NeighborSuccess SuccessRule         `json:"neighbor_success,omitempty"`
	BroaderCommands []BroaderValidation `json:"broader_commands,omitempty"`
}

type Phase struct {
	Command string      `json:"command"`
	Repeat  int         `json:"repeat,omitempty"`
	Success SuccessRule `json:"success"`
}

type RunContract struct {
	Airlock      base.Contract  `json:"airlock"`
	Objective    string         `json:"objective"`
	Mode         string         `json:"mode"`
	TargetPath   string         `json:"targetPath,omitempty"`
	ArtifactsDir string         `json:"artifactsDir,omitempty"`
	Setup        []SetupStep    `json:"setup,omitempty"`
	Baseline     *Phase         `json:"baseline,omitempty"`
	Reproduction Phase          `json:"reproduction"`
	Patches      []PatchStep    `json:"patches,omitempty"`
	Validation   ValidationSpec `json:"validation"`
	Safety       SafetyBudget   `json:"safety,omitempty"`
	Stuck        StuckPolicy    `json:"stuck,omitempty"`
	Campaign     *CampaignSpec  `json:"campaign,omitempty"`
}

func LoadRunContract(path string) (RunContract, error) {
	var c RunContract
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func ValidateRunContract(c RunContract) []string {
	errs := base.Validate(c.Airlock)
	if c.Objective == "" {
		errs = append(errs, "objective is required")
	}
	if c.Mode != "read_only" && c.Mode != "mutate" {
		errs = append(errs, "mode must be read_only or mutate")
	}
	if c.Reproduction.Command == "" {
		errs = append(errs, "reproduction.command is required")
	}
	if c.Validation.TargetCommand == "" {
		errs = append(errs, "validation.target_command is required")
	}
	if c.Mode == "mutate" && len(c.Patches) == 0 {
		errs = append(errs, "patches are required in mutate mode")
	}
	return errs
}
