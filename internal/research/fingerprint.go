package research

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var failPattern = regexp.MustCompile(`--- FAIL: ([^\n]+)`)
var pkgFailPattern = regexp.MustCompile(`(?m)^FAIL[ \t]+([^\s]+)`)
var panicPattern = regexp.MustCompile(`panic: ([^\n]+)`)
var runtimePattern = regexp.MustCompile(`runtime error: ([^\n]+)`)

func ExtractFailureSignatures(result CommandResult) []string {
	if result.ExitCode == 0 {
		return nil
	}
	combined := result.Stdout + "\n" + result.Stderr
	set := map[string]struct{}{}
	for _, m := range failPattern.FindAllStringSubmatch(combined, -1) {
		set["test_failure:"+strings.TrimSpace(m[1])] = struct{}{}
	}
	for _, m := range pkgFailPattern.FindAllStringSubmatch(combined, -1) {
		set["package_failure:"+strings.TrimSpace(m[1])] = struct{}{}
	}
	for _, m := range panicPattern.FindAllStringSubmatch(combined, -1) {
		set["panic:"+strings.TrimSpace(m[1])] = struct{}{}
	}
	for _, m := range runtimePattern.FindAllStringSubmatch(combined, -1) {
		set["panic:"+strings.TrimSpace(m[1])] = struct{}{}
	}
	if len(set) == 0 {
		line := ""
		for _, candidate := range strings.Split(combined, "\n") {
			candidate = strings.TrimSpace(candidate)
			if candidate != "" {
				line = candidate
				break
			}
		}
		if line == "" {
			line = fmt.Sprintf("exit:%d", result.ExitCode)
		}
		set["command_failure:"+line] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for sig := range set {
		out = append(out, sig)
	}
	sort.Strings(out)
	return out
}

func SummarizeFailures(results []CommandResult) []FailureFingerprint {
	m := map[string]*FailureFingerprint{}
	for _, result := range results {
		for _, sig := range ExtractFailureSignatures(result) {
			entry := m[sig]
			if entry == nil {
				kind := sig
				if idx := strings.Index(sig, ":"); idx >= 0 {
					kind = sig[:idx]
				}
				entry = &FailureFingerprint{Kind: kind, Signature: sig}
				m[sig] = entry
			}
			entry.Count++
			if len(entry.Samples) < 3 {
				entry.Samples = append(entry.Samples, result.Command+" :: "+sig)
			}
		}
	}
	out := make([]FailureFingerprint, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Signature < out[j].Signature
		}
		return out[i].Count > out[j].Count
	})
	return out
}
