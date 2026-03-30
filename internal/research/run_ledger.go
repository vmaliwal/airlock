package research

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vmaliwal/airlock/internal/util"
)

const (
	MetricsDirEnv     = "AIRLOCK_METRICS_DIR"
	CustomerIDEnv     = "AIRLOCK_CUSTOMER_ID"
	defaultCustomerID = "default"
)

type RunSummary struct {
	RunID                   string `json:"run_id"`
	Timestamp               string `json:"timestamp"`
	CustomerID              string `json:"customer_id"`
	RepoKey                 string `json:"repo_key"`
	IssueKey                string `json:"issue_key,omitempty"`
	Entrypoint              string `json:"entrypoint"`
	AirlockVersion          string `json:"airlock_version"`
	Backend                 string `json:"backend,omitempty"`
	RepoSHA                 string `json:"repo_sha,omitempty"`
	ReproStatus             string `json:"repro_status,omitempty"`
	Advance                 bool   `json:"advance"`
	CredibleAdvancement     bool   `json:"credible_advancement"`
	VerifiedIssueResolution bool   `json:"verified_issue_resolution"`
	FixConfidence           string `json:"fix_confidence,omitempty"`
	ValidationScope         string `json:"validation_scope,omitempty"`
	AttemptCount            int    `json:"attempt_count,omitempty"`
	RoundCount              int    `json:"round_count,omitempty"`
	DurationSeconds         int64  `json:"duration_seconds,omitempty"`
	FailureCategory         string `json:"failure_category,omitempty"`
	WinningAttempt          string `json:"winning_attempt,omitempty"`
}

type MetricRollup struct {
	Runs                         int     `json:"runs"`
	CredibleAdvancementCount     int     `json:"credible_advancement_count"`
	VerifiedIssueResolutionCount int     `json:"verified_issue_resolution_count"`
	CredibleAdvancementRate      float64 `json:"credible_advancement_rate"`
	VerifiedIssueResolutionRate  float64 `json:"verified_issue_resolution_rate"`
}

type MetricsSummary struct {
	LedgerPath string                  `json:"ledger_path,omitempty"`
	Global     MetricRollup            `json:"global"`
	ByRepo     map[string]MetricRollup `json:"by_repo"`
	ByCustomer map[string]MetricRollup `json:"by_customer"`
}

func NewRunID(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "run"
	}
	return fmt.Sprintf("%s-%d", util.SafeName(prefix), time.Now().UTC().UnixNano())
}

func CurrentCustomerID() string {
	if v := strings.TrimSpace(os.Getenv(CustomerIDEnv)); v != "" {
		return v
	}
	return defaultCustomerID
}

func DefaultMetricsDir() string {
	if v := strings.TrimSpace(os.Getenv(MetricsDirEnv)); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", ".airlock", "metrics")
	}
	return filepath.Join(home, ".airlock", "metrics")
}

func DefaultRunLedgerPath() string {
	return filepath.Join(DefaultMetricsDir(), "runs.jsonl")
}

func AppendRunSummary(summary RunSummary) error {
	if strings.TrimSpace(summary.RunID) == "" {
		summary.RunID = NewRunID(summary.Entrypoint)
	}
	if strings.TrimSpace(summary.Timestamp) == "" {
		summary.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(summary.CustomerID) == "" {
		summary.CustomerID = CurrentCustomerID()
	}
	if strings.TrimSpace(summary.AirlockVersion) == "" {
		summary.AirlockVersion = AirlockVersion()
	}
	path := DefaultRunLedgerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(summary)
}

func LoadRunSummaries(path string) ([]RunSummary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	items := []RunSummary{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var item RunSummary
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func SummarizeRunMetrics(items []RunSummary) MetricsSummary {
	out := MetricsSummary{
		ByRepo:     map[string]MetricRollup{},
		ByCustomer: map[string]MetricRollup{},
	}
	for _, item := range items {
		out.Global = addMetricRollup(out.Global, item)
		if item.RepoKey != "" {
			out.ByRepo[item.RepoKey] = addMetricRollup(out.ByRepo[item.RepoKey], item)
		}
		customer := item.CustomerID
		if customer == "" {
			customer = defaultCustomerID
		}
		out.ByCustomer[customer] = addMetricRollup(out.ByCustomer[customer], item)
	}
	out.Global = finalizeMetricRollup(out.Global)
	for key, value := range out.ByRepo {
		out.ByRepo[key] = finalizeMetricRollup(value)
	}
	for key, value := range out.ByCustomer {
		out.ByCustomer[key] = finalizeMetricRollup(value)
	}
	out.ByRepo = sortedMetricMap(out.ByRepo)
	out.ByCustomer = sortedMetricMap(out.ByCustomer)
	return out
}

func addMetricRollup(m MetricRollup, item RunSummary) MetricRollup {
	m.Runs++
	if item.CredibleAdvancement {
		m.CredibleAdvancementCount++
	}
	if item.VerifiedIssueResolution {
		m.VerifiedIssueResolutionCount++
	}
	return m
}

func finalizeMetricRollup(m MetricRollup) MetricRollup {
	if m.Runs > 0 {
		m.CredibleAdvancementRate = float64(m.CredibleAdvancementCount) / float64(m.Runs)
		m.VerifiedIssueResolutionRate = float64(m.VerifiedIssueResolutionCount) / float64(m.Runs)
	}
	return m
}

func sortedMetricMap(in map[string]MetricRollup) map[string]MetricRollup {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]MetricRollup, len(in))
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}

func NormalizeRepoKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	normalized := strings.TrimSuffix(NormalizeCloneURL(raw), ".git")
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "ssh://")
	if strings.HasPrefix(normalized, "github.com/") {
		return strings.TrimPrefix(normalized, "github.com/")
	}
	if strings.HasPrefix(normalized, "git@github.com:") {
		return strings.TrimPrefix(strings.TrimSuffix(normalized, ".git"), "git@github.com:")
	}
	return strings.Trim(normalized, "/")
}

func RepoKeyForPath(path string) string {
	if remote, err := GitRemoteOrigin(path); err == nil {
		if key := NormalizeRepoKey(remote); key != "" {
			return key
		}
	}
	if top, err := GitTopLevel(path); err == nil {
		return filepath.Base(top)
	}
	return filepath.Base(path)
}
