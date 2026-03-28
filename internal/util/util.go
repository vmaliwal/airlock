package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RunOptions struct {
	Cwd     string
	Env     []string
	Timeout time.Duration
}

func SafeName(input string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := strings.ToLower(input)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		return "airlock"
	}
	return s
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func WriteFile(path string, data []byte, mode os.FileMode) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func EnvMapFromSlice(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func RequireAbsolute(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func RunLocal(command string, args []string, opts RunOptions) ([]byte, error) {
	cmd := exec.Command(command, args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}
	if opts.Timeout > 0 {
		timer := time.AfterFunc(opts.Timeout, func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
		defer timer.Stop()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w\n%s", command, args, err, string(out))
	}
	return out, nil
}
