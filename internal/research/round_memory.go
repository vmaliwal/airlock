package research

import (
	"sort"
	"strings"
)

func BuildNextRoundPlanInput(base PlanInput, previous *AutofixSummary) PlanInput {
	if previous == nil {
		return base
	}
	failedAttempts := []string{}
	failedKinds := []string{}
	fingerprints := []string{}
	for _, attempt := range previous.Attempts {
		if attempt.Success {
			continue
		}
		if attempt.Name != "" {
			failedAttempts = append(failedAttempts, attempt.Name)
		}
		if attempt.MutationKind != "" {
			failedKinds = append(failedKinds, attempt.MutationKind)
		}
		for _, fp := range attempt.ValidationFingerprints {
			if fp.Signature != "" {
				fingerprints = append(fingerprints, fp.Signature)
			}
		}
	}
	failedAttempts = dedupeAndSort(failedAttempts)
	failedKinds = dedupeAndSort(failedKinds)
	fingerprints = dedupeAndSort(fingerprints)
	memory := []string{"## Prior round memory"}
	if len(failedAttempts) > 0 {
		memory = append(memory, "failed_attempts: "+strings.Join(failedAttempts, ", "))
	}
	if len(failedKinds) > 0 {
		memory = append(memory, "failed_mutation_kinds: "+strings.Join(failedKinds, ", "))
	}
	if len(fingerprints) > 0 {
		memory = append(memory, "failed_fingerprints: "+strings.Join(fingerprints, ", "))
	}
	memory = append(memory, "instruction: avoid repeating previously failed exact attempt shapes unless new evidence suggests a materially different variant")
	out := base
	if strings.TrimSpace(out.Notes) != "" {
		out.Notes = strings.TrimSpace(out.Notes) + "\n\n" + strings.Join(memory, "\n")
	} else {
		out.Notes = strings.Join(memory, "\n")
	}
	return out
}

func dedupeAndSort(items []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
