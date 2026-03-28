package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type PlanInput struct {
	RepoPath       string `json:"repoPath"`
	IssueURL       string `json:"issueUrl,omitempty"`
	FailingCommand string `json:"failingCommand,omitempty"`
	FailureText    string `json:"failureText,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

func LoadPlanInput(path string) (PlanInput, error) {
	var in PlanInput
	data, err := os.ReadFile(path)
	if err != nil {
		return in, err
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return in, err
	}
	if in.RepoPath != "" && !filepath.IsAbs(in.RepoPath) {
		in.RepoPath = filepath.Join(filepath.Dir(path), in.RepoPath)
	}
	return in, nil
}

func ResolvePlanInput(arg string) (PlanInput, error) {
	if strings.HasSuffix(arg, ".json") {
		return LoadPlanInput(arg)
	}
	return PlanInput{RepoPath: arg}, nil
}
