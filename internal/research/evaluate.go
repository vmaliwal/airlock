package research

import (
	"fmt"
	"strings"
)

func EvaluateSingle(result CommandResult, rule SuccessRule) EvaluationResult {
	observed := []string{}
	failed := []string{}
	if rule.ExitCode != nil {
		if result.ExitCode == *rule.ExitCode {
			observed = append(observed, fmt.Sprintf("exit_code == %d", *rule.ExitCode))
		} else {
			failed = append(failed, fmt.Sprintf("exit_code != %d", *rule.ExitCode))
		}
	}
	for _, s := range rule.StdoutContains {
		if strings.Contains(result.Stdout, s) {
			observed = append(observed, "stdout_contains:"+s)
		} else {
			failed = append(failed, "missing_stdout:"+s)
		}
	}
	for _, s := range rule.StderrNotContains {
		if !strings.Contains(result.Stderr, s) {
			observed = append(observed, "stderr_not_contains:"+s)
		} else {
			failed = append(failed, "stderr_contains:"+s)
		}
	}
	if rule.MaxDurationMs != nil {
		if result.DurationMs <= *rule.MaxDurationMs {
			observed = append(observed, fmt.Sprintf("duration <= %d", *rule.MaxDurationMs))
		} else {
			failed = append(failed, fmt.Sprintf("duration > %d", *rule.MaxDurationMs))
		}
	}
	return EvaluationResult{Passed: len(failed) == 0, Observed: observed, FailedRules: failed}
}

func EvaluateRepeated(results []CommandResult, rule SuccessRule) EvaluationResult {
	passCount := 0
	for _, result := range results {
		if result.ExitCode == 0 {
			passCount++
		}
	}
	failCount := len(results) - passCount
	passRate := 0.0
	if len(results) > 0 {
		passRate = float64(passCount) / float64(len(results))
	}
	observed := []string{fmt.Sprintf("pass_rate=%v", passRate), fmt.Sprintf("pass_count=%d", passCount), fmt.Sprintf("fail_count=%d", failCount)}
	failed := []string{}
	if rule.ExitCode != nil {
		mismatches := 0
		for _, result := range results {
			if result.ExitCode != *rule.ExitCode {
				mismatches++
			}
		}
		if mismatches == 0 {
			observed = append(observed, fmt.Sprintf("all_exit_code == %d", *rule.ExitCode))
		} else {
			failed = append(failed, fmt.Sprintf("exit_code mismatches: %d", mismatches))
		}
	}
	if rule.MinPassRate != nil && passRate < *rule.MinPassRate {
		failed = append(failed, fmt.Sprintf("pass_rate < %v", *rule.MinPassRate))
	}
	if rule.MinFailures != nil && failCount < *rule.MinFailures {
		failed = append(failed, fmt.Sprintf("fail_count < %d", *rule.MinFailures))
	}
	if rule.MaxFailures != nil && failCount > *rule.MaxFailures {
		failed = append(failed, fmt.Sprintf("fail_count > %d", *rule.MaxFailures))
	}
	return EvaluationResult{Passed: len(failed) == 0, PassRate: passRate, PassCount: passCount, FailCount: failCount, Observed: observed, FailedRules: failed}
}
