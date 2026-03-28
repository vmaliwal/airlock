package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SearchReplaceMutation struct {
	Path    string `json:"path"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

type InsertAfterMutation struct {
	Path       string `json:"path"`
	AnchorText string `json:"anchorText"`
	InsertText string `json:"insertText"`
}

type ReplaceLineMutation struct {
	Path    string `json:"path"`
	OldLine string `json:"oldLine"`
	NewLine string `json:"newLine"`
}

type CreateFileMutation struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ApplyPatchMutation struct {
	Patch string `json:"patch"`
}

type EnsureLineMutation struct {
	Path string `json:"path"`
	Line string `json:"line"`
}

type MutationSpec struct {
	SearchReplace *SearchReplaceMutation `json:"search_replace,omitempty"`
	InsertAfter   *InsertAfterMutation   `json:"insert_after,omitempty"`
	ReplaceLine   *ReplaceLineMutation   `json:"replace_line,omitempty"`
	CreateFile    *CreateFileMutation    `json:"create_file,omitempty"`
	ApplyPatch    *ApplyPatchMutation    `json:"apply_patch,omitempty"`
	EnsureLine    *EnsureLineMutation    `json:"ensure_line,omitempty"`
}

type AttemptFile struct {
	Repo         string       `json:"repo"`
	ArtifactsDir string       `json:"artifactsDir"`
	Checkpoint   string       `json:"checkpoint,omitempty"`
	Attempt      AttemptSpec  `json:"attempt"`
	Mutation     MutationSpec `json:"mutation,omitempty"`
}

type LessonRecord struct {
	Timestamp    string               `json:"timestamp"`
	Repo         string               `json:"repo"`
	AttemptName  string               `json:"attemptName"`
	MutationKind string               `json:"mutationKind,omitempty"`
	Success      bool                 `json:"success"`
	BudgetErrors []string             `json:"budgetErrors,omitempty"`
	Fingerprints []FailureFingerprint `json:"fingerprints,omitempty"`
	PatchPath    string               `json:"patchPath,omitempty"`
}

func LoadAttemptFile(path string) (AttemptFile, error) {
	var c AttemptFile
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.Repo != "" && !filepath.IsAbs(c.Repo) {
		c.Repo = filepath.Join(filepath.Dir(path), c.Repo)
	}
	if c.ArtifactsDir != "" && !filepath.IsAbs(c.ArtifactsDir) {
		c.ArtifactsDir = filepath.Join(filepath.Dir(path), c.ArtifactsDir)
	}
	return c, nil
}

func ValidateAttemptFile(c AttemptFile) []string {
	errs := []string{}
	if c.Repo == "" {
		errs = append(errs, "repo is required")
	}
	if c.ArtifactsDir == "" {
		errs = append(errs, "artifactsDir is required")
	}
	if c.Attempt.Name == "" {
		errs = append(errs, "attempt.name is required")
	}
	if c.Attempt.Validation.Command == "" {
		errs = append(errs, "attempt.validation.command is required")
	}
	if c.Attempt.MutationCommand == "" && c.Mutation.SearchReplace == nil && c.Mutation.InsertAfter == nil && c.Mutation.ReplaceLine == nil && c.Mutation.CreateFile == nil && c.Mutation.ApplyPatch == nil && c.Mutation.EnsureLine == nil {
		errs = append(errs, "attempt requires mutationCommand or supported mutation spec")
	}
	return errs
}

func ApplyMutationSpec(repo string, m MutationSpec) (CommandResult, error) {
	if m.SearchReplace != nil {
		path := filepath.Join(repo, m.SearchReplace.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return CommandResult{Command: "search_replace", ExitCode: 1, Stderr: err.Error()}, nil
		}
		s := string(data)
		if !containsText(s, m.SearchReplace.OldText) {
			return CommandResult{Command: "search_replace", ExitCode: 1, Stderr: "oldText not found"}, nil
		}
		s = replaceOnce(s, m.SearchReplace.OldText, m.SearchReplace.NewText)
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			return CommandResult{Command: "search_replace", ExitCode: 1, Stderr: err.Error()}, nil
		}
		return CommandResult{Command: "search_replace", ExitCode: 0}, nil
	}
	if m.InsertAfter != nil {
		path := filepath.Join(repo, m.InsertAfter.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return CommandResult{Command: "insert_after", ExitCode: 1, Stderr: err.Error()}, nil
		}
		s := string(data)
		if !containsText(s, m.InsertAfter.AnchorText) {
			return CommandResult{Command: "insert_after", ExitCode: 1, Stderr: "anchorText not found"}, nil
		}
		s = strings.Replace(s, m.InsertAfter.AnchorText, m.InsertAfter.AnchorText+m.InsertAfter.InsertText, 1)
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			return CommandResult{Command: "insert_after", ExitCode: 1, Stderr: err.Error()}, nil
		}
		return CommandResult{Command: "insert_after", ExitCode: 0}, nil
	}
	if m.ReplaceLine != nil {
		path := filepath.Join(repo, m.ReplaceLine.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return CommandResult{Command: "replace_line", ExitCode: 1, Stderr: err.Error()}, nil
		}
		lines := strings.Split(string(data), "\n")
		found := false
		for i, line := range lines {
			if line == m.ReplaceLine.OldLine {
				lines[i] = m.ReplaceLine.NewLine
				found = true
				break
			}
		}
		if !found {
			return CommandResult{Command: "replace_line", ExitCode: 1, Stderr: "oldLine not found"}, nil
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return CommandResult{Command: "replace_line", ExitCode: 1, Stderr: err.Error()}, nil
		}
		return CommandResult{Command: "replace_line", ExitCode: 0}, nil
	}
	if m.CreateFile != nil {
		path := filepath.Join(repo, m.CreateFile.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return CommandResult{Command: "create_file", ExitCode: 1, Stderr: err.Error()}, nil
		}
		if err := os.WriteFile(path, []byte(m.CreateFile.Content), 0o644); err != nil {
			return CommandResult{Command: "create_file", ExitCode: 1, Stderr: err.Error()}, nil
		}
		return CommandResult{Command: "create_file", ExitCode: 0}, nil
	}
	if m.ApplyPatch != nil {
		patchPath := filepath.Join(repo, ".airlock-inline.patch")
		if err := os.WriteFile(patchPath, []byte(m.ApplyPatch.Patch), 0o644); err != nil {
			return CommandResult{Command: "apply_patch", ExitCode: 1, Stderr: err.Error()}, nil
		}
		res, err := RunLocalCommand(repo, "git apply .airlock-inline.patch", 30*time.Second)
		_ = os.Remove(patchPath)
		if err != nil {
			return CommandResult{Command: "apply_patch", ExitCode: 1, Stderr: err.Error()}, nil
		}
		res.Command = "apply_patch"
		return res, nil
	}
	if m.EnsureLine != nil {
		path := filepath.Join(repo, m.EnsureLine.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return CommandResult{Command: "ensure_line", ExitCode: 1, Stderr: err.Error()}, nil
		}
		content := string(data)
		needle := m.EnsureLine.Line
		if strings.Contains(content, needle) {
			return CommandResult{Command: "ensure_line", ExitCode: 0}, nil
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += needle
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return CommandResult{Command: "ensure_line", ExitCode: 1, Stderr: err.Error()}, nil
		}
		return CommandResult{Command: "ensure_line", ExitCode: 0}, nil
	}
	return CommandResult{Command: "mutation", ExitCode: 1, Stderr: "no supported mutation spec provided"}, nil
}

func MutationKind(m MutationSpec, attempt AttemptSpec) string {
	if m.SearchReplace != nil {
		return "search_replace"
	}
	if m.InsertAfter != nil {
		return "insert_after"
	}
	if m.ReplaceLine != nil {
		return "replace_line"
	}
	if m.CreateFile != nil {
		return "create_file"
	}
	if m.ApplyPatch != nil {
		return "apply_patch"
	}
	if m.EnsureLine != nil {
		return "ensure_line"
	}
	if attempt.MutationCommand != "" {
		return "command"
	}
	return "unknown"
}

func RunAttemptFile(c AttemptFile) (AttemptOutcome, error) {
	if err := os.MkdirAll(c.ArtifactsDir, 0o755); err != nil {
		return AttemptOutcome{}, err
	}
	checkpoint := c.Checkpoint
	if checkpoint == "" {
		sha, err := GitHeadSHA(c.Repo)
		if err != nil {
			return AttemptOutcome{}, err
		}
		checkpoint = sha
	}
	attempt := c.Attempt
	attempt.Timeout = 30 * time.Second
	if c.Mutation.SearchReplace != nil || c.Mutation.InsertAfter != nil || c.Mutation.ReplaceLine != nil || c.Mutation.CreateFile != nil || c.Mutation.ApplyPatch != nil || c.Mutation.EnsureLine != nil {
		attempt.MutationCommand = ""
	}
	outcome, err := RunNativeAttemptWithMutation(c.Repo, c.ArtifactsDir, checkpoint, attempt, c.Mutation)
	if err != nil {
		return outcome, err
	}
	_ = AppendLesson(filepath.Join(c.ArtifactsDir, "lessons.jsonl"), LessonRecord{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Repo:         c.Repo,
		AttemptName:  outcome.Name,
		MutationKind: MutationKind(c.Mutation, c.Attempt),
		Success:      outcome.Success,
		BudgetErrors: outcome.BudgetErrors,
		Fingerprints: outcome.ValidationFingerprints,
		PatchPath:    outcome.PatchPath,
	})
	return outcome, nil
}

func AppendLesson(path string, lesson LessonRecord) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(lesson)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func containsText(s, sub string) bool { return strings.Contains(s, sub) }
func replaceOnce(s, oldText, newText string) string {
	return strings.Replace(s, oldText, newText, 1)
}
