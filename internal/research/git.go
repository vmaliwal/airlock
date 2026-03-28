package research

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vmaliwal/airlock/internal/util"
)

type GitDiffStats struct {
	ChangedFiles      []string `json:"changedFiles"`
	FilesChangedCount int      `json:"filesChangedCount"`
	LocChanged        int      `json:"locChanged"`
}

func GitTopLevel(repo string) (string, error) {
	out, err := util.RunLocal("git", []string{"rev-parse", "--show-toplevel"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GitRemoteOrigin(repo string) (string, error) {
	out, err := util.RunLocal("git", []string{"remote", "get-url", "origin"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GitHeadSHA(repo string) (string, error) {
	out, err := util.RunLocal("git", []string{"rev-parse", "HEAD"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GitIsDirty(repo string) (bool, error) {
	out, err := util.RunLocal("git", []string{"status", "--porcelain"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func GitDiffNumstat(repo string) (GitDiffStats, error) {
	out, err := util.RunLocal("git", []string{"diff", "--numstat", "HEAD"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return GitDiffStats{}, err
	}
	stats := GitDiffStats{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		stats.ChangedFiles = append(stats.ChangedFiles, parts[2])
		if parts[0] != "-" {
			v, err := strconv.Atoi(parts[0])
			if err != nil {
				return GitDiffStats{}, err
			}
			stats.LocChanged += v
		}
		if parts[1] != "-" {
			v, err := strconv.Atoi(parts[1])
			if err != nil {
				return GitDiffStats{}, err
			}
			stats.LocChanged += v
		}
	}
	stats.FilesChangedCount = len(stats.ChangedFiles)
	return stats, nil
}

func GitWritePatch(repo, outPath string) error {
	patch, err := util.RunLocal("git", []string{"diff", "--binary", "HEAD"}, util.RunOptions{Cwd: repo})
	if err != nil {
		return err
	}
	return util.WriteFile(outPath, patch, 0o644)
}

func GitResetHard(repo string) error {
	return GitResetHardTo(repo, "HEAD")
}

func GitResetHardTo(repo, rev string) error {
	_, err := util.RunLocal("git", []string{"reset", "--hard", rev}, util.RunOptions{Cwd: repo})
	return err
}

func GitClean(repo string) error {
	return GitCleanExcept(repo, nil)
}

func GitCleanExcept(repo string, excludes []string) error {
	args := []string{"clean", "-fd"}
	for _, ex := range excludes {
		args = append(args, "-e", ex)
	}
	_, err := util.RunLocal("git", args, util.RunOptions{Cwd: repo})
	return err
}

func GitResetAttempt(repo string) error {
	return GitResetAttemptExcept(repo, nil)
}

func GitResetAttemptExcept(repo string, excludes []string) error {
	if err := GitResetHard(repo); err != nil {
		return err
	}
	return GitCleanExcept(repo, excludes)
}

func GitEnsureIdentity(repo string) error {
	if _, err := util.RunLocal("git", []string{"config", "user.email", "airlock@example.com"}, util.RunOptions{Cwd: repo}); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"config", "user.name", "Airlock"}, util.RunOptions{Cwd: repo}); err != nil {
		return err
	}
	return nil
}

func GitCommitAll(repo, message string) error {
	if err := GitEnsureIdentity(repo); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"add", "-A"}, util.RunOptions{Cwd: repo}); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"commit", "-m", message}, util.RunOptions{Cwd: repo}); err != nil {
		return err
	}
	return nil
}

func InitTempGitRepo(root string, files map[string]string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"init"}, util.RunOptions{Cwd: root}); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"config", "user.email", "airlock@example.com"}, util.RunOptions{Cwd: root}); err != nil {
		return err
	}
	if _, err := util.RunLocal("git", []string{"config", "user.name", "Airlock"}, util.RunOptions{Cwd: root}); err != nil {
		return err
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := util.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := GitCommitAll(root, "initial"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}
	return nil
}
