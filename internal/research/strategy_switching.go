package research

import "strings"

func extractFailedMutationKindsFromNotes(notes string) []string {
	for _, line := range strings.Split(notes, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "failed_mutation_kinds:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "failed_mutation_kinds:"))
		parts := strings.Split(rest, ",")
		return dedupeAndSort(parts)
	}
	return nil
}

func avoidMutationKinds(input PlanInput) map[string]struct{} {
	items := extractFailedMutationKindsFromNotes(input.Notes)
	out := map[string]struct{}{}
	for _, item := range items {
		if item != "" {
			out[item] = struct{}{}
		}
	}
	return out
}

func filterAllowedMutations(allowed []string, avoid map[string]struct{}) []string {
	if len(avoid) == 0 {
		return dedupeStrings(allowed)
	}
	filtered := []string{}
	for _, item := range allowed {
		if _, blocked := avoid[item]; blocked {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return dedupeStrings(allowed)
	}
	return dedupeStrings(filtered)
}

func filterRankedMutationKinds(items []MutationKindScore, avoid map[string]struct{}) []MutationKindScore {
	if len(avoid) == 0 {
		return items
	}
	preferred := []MutationKindScore{}
	deferred := []MutationKindScore{}
	for _, item := range items {
		if _, blocked := avoid[item.Kind]; blocked {
			deferred = append(deferred, item)
			continue
		}
		preferred = append(preferred, item)
	}
	if len(preferred) == 0 {
		return items
	}
	return append(preferred, deferred...)
}

func filterSynthesizedAttempts(attempts []SynthesizedAttempt, avoid map[string]struct{}) []SynthesizedAttempt {
	if len(avoid) == 0 {
		return attempts
	}
	filtered := []SynthesizedAttempt{}
	for _, attempt := range attempts {
		if _, blocked := avoid[attempt.MutationKind]; blocked {
			continue
		}
		filtered = append(filtered, attempt)
	}
	if len(filtered) == 0 {
		return attempts
	}
	return filtered
}
