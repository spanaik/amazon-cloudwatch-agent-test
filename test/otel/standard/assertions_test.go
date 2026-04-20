//go:build integration

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package standard

import (
	"fmt"
	"sort"
	"testing"

	"github.com/aws/amazon-cloudwatch-agent-test/util/otelmetrics"
)

func requireEqual(t *testing.T, got, want any, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: got %v, want %v", fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...), got, want)
		}
		t.Fatalf("got %v, want %v", got, want)
	}
}

func requireTrue(t *testing.T, condition bool, msgAndArgs ...any) {
	t.Helper()
	if !condition {
		if len(msgAndArgs) > 0 {
			t.Fatalf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		t.Fatalf("expected true, got false")
	}
}

func requireNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: %v", fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...), err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireNonEmpty(t *testing.T, results []otelmetrics.MetricResult, msgAndArgs ...any) {
	t.Helper()
	if len(results) == 0 {
		if len(msgAndArgs) > 0 {
			t.Fatalf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		t.Fatalf("expected non-empty results, got 0")
	}
}

func filterByHostType(results []otelmetrics.MetricResult, instanceType string) []otelmetrics.MetricResult {
	var out []otelmetrics.MetricResult
	for _, r := range results {
		if r.Labels.Resource["host.type"] == instanceType {
			out = append(out, r)
		}
	}
	return out
}

func uniqueValues(results []otelmetrics.MetricResult, attr string) []string {
	seen := make(map[string]struct{})
	for _, r := range results {
		if v, ok := r.Labels.Resource[attr]; ok && v != "" {
			seen[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func uniqueDatapointValues(results []otelmetrics.MetricResult, attr string) []string {
	seen := make(map[string]struct{})
	for _, r := range results {
		if v, ok := r.Labels.Datapoint[attr]; ok && v != "" {
			seen[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func uniqueAnyValues(results []otelmetrics.MetricResult, attr string) []string {
	seen := make(map[string]struct{})
	for _, r := range results {
		if v, ok := r.Labels.Resource[attr]; ok && v != "" {
			seen[v] = struct{}{}
		} else if v, ok := r.Labels.Datapoint[attr]; ok && v != "" {
			seen[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func getAnyValue(r otelmetrics.MetricResult, attr string) string {
	if v, ok := r.Labels.Resource[attr]; ok && v != "" {
		return v
	}
	return r.Labels.Datapoint[attr]
}
