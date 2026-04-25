package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Comparison struct {
	Baseline  *Summary
	Candidate *Summary
	Endpoints []EndpointComparison
}

type EndpointComparison struct {
	Name      string
	Status    string
	Baseline  Endpoint
	Candidate Endpoint
}

func LoadSummaryFile(path string) (*Summary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read summary: %w", err)
	}
	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse summary JSON: %w", err)
	}
	if summary.Endpoints == nil {
		summary.Endpoints = map[string]Endpoint{}
	}
	return &summary, nil
}

func CompareSummaries(baseline *Summary, candidate *Summary) *Comparison {
	if baseline.Endpoints == nil {
		baseline.Endpoints = map[string]Endpoint{}
	}
	if candidate.Endpoints == nil {
		candidate.Endpoints = map[string]Endpoint{}
	}
	names := map[string]bool{}
	for name := range baseline.Endpoints {
		names[name] = true
	}
	for name := range candidate.Endpoints {
		names[name] = true
	}
	endpoints := make([]EndpointComparison, 0, len(names))
	for name := range names {
		base, hasBase := baseline.Endpoints[name]
		next, hasCandidate := candidate.Endpoints[name]
		status := "changed"
		switch {
		case hasBase && !hasCandidate:
			status = "removed"
		case !hasBase && hasCandidate:
			status = "added"
		}
		endpoints = append(endpoints, EndpointComparison{
			Name:      name,
			Status:    status,
			Baseline:  base,
			Candidate: next,
		})
	}
	sort.Slice(endpoints, func(i, j int) bool {
		left := endpoints[i]
		right := endpoints[j]
		leftRank := comparisonStatusRank(left.Status)
		rightRank := comparisonStatusRank(right.Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftP95Delta := left.Candidate.P95MS - left.Baseline.P95MS
		rightP95Delta := right.Candidate.P95MS - right.Baseline.P95MS
		if leftP95Delta != rightP95Delta {
			return leftP95Delta > rightP95Delta
		}
		return left.Name < right.Name
	})
	return &Comparison{Baseline: baseline, Candidate: candidate, Endpoints: endpoints}
}

func RenderComparisonMarkdown(c *Comparison) string {
	var b strings.Builder
	b.WriteString("# Loadwright Comparison\n\n")
	b.WriteString("## Overview\n\n")
	b.WriteString("| Metric | Baseline | Candidate | Delta |\n")
	b.WriteString("| --- | ---: | ---: | ---: |\n")
	writeIntMetric(&b, "Total samples", c.Baseline.TotalSamples, c.Candidate.TotalSamples)
	writeIntMetric(&b, "Failed samples", c.Baseline.Failed, c.Candidate.Failed)
	writeFloatMetric(&b, "Error rate", c.Baseline.ErrorRate, c.Candidate.ErrorRate, "%")
	writeFloatMetric(&b, "Average", c.Baseline.AverageMS, c.Candidate.AverageMS, " ms")
	writeFloatMetric(&b, "p95", c.Baseline.P95MS, c.Candidate.P95MS, " ms")
	writeFloatMetric(&b, "p99", c.Baseline.P99MS, c.Candidate.P99MS, " ms")

	b.WriteString("\n## Endpoints\n\n")
	if len(c.Endpoints) == 0 {
		b.WriteString("No endpoint data found.\n")
		return b.String()
	}
	b.WriteString("| Endpoint | Status | Baseline failed | Candidate failed | Error rate delta | Average delta | p95 delta |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, endpoint := range c.Endpoints {
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %s | %s | %s |\n",
			escapeMarkdownCell(endpoint.Name),
			endpoint.Status,
			endpoint.Baseline.Failed,
			endpoint.Candidate.Failed,
			formatSignedFloat(endpointErrorRate(endpoint.Candidate)-endpointErrorRate(endpoint.Baseline), "%"),
			formatSignedFloat(endpoint.Candidate.AverageMS-endpoint.Baseline.AverageMS, " ms"),
			formatSignedFloat(endpoint.Candidate.P95MS-endpoint.Baseline.P95MS, " ms"),
		)
	}
	return b.String()
}

func writeIntMetric(b *strings.Builder, name string, baseline int, candidate int) {
	fmt.Fprintf(b, "| %s | %d | %d | %+d |\n", name, baseline, candidate, candidate-baseline)
}

func writeFloatMetric(b *strings.Builder, name string, baseline float64, candidate float64, suffix string) {
	fmt.Fprintf(b, "| %s | %.2f%s | %.2f%s | %s |\n", name, baseline, suffix, candidate, suffix, formatSignedFloat(candidate-baseline, suffix))
}

func formatSignedFloat(value float64, suffix string) string {
	return fmt.Sprintf("%+.2f%s", value, suffix)
}

func comparisonStatusRank(status string) int {
	switch status {
	case "changed":
		return 0
	case "added":
		return 1
	case "removed":
		return 2
	default:
		return 3
	}
}
