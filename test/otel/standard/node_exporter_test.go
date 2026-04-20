//go:build integration

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package standard

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/amazon-cloudwatch-agent-test/util/otelmetrics"
)

var nodeExporterMetricNames = metricNames(nodeExporterMetrics)

func TestNodeExporterInstrumentationSource(t *testing.T) {
	for _, metricName := range nodeExporterMetricNames {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				name, ok := r.Labels.Instrumentation["@name"]
				requireTrue(t, ok, "%s missing @instrumentation.@name", metricName)
				requireEqual(t, name, "github.com/prometheus/node_exporter",
					"%s instrumentation name", metricName)
			}
		})
	}
}

func TestNodeExporterInstrumentationConsistent(t *testing.T) {
	for _, metricName := range nodeExporterMetricNames {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			names := make(map[string]struct{})
			for _, r := range results {
				if n, ok := r.Labels.Instrumentation["@name"]; ok {
					names[n] = struct{}{}
				}
			}
			requireEqual(t, len(names), 1,
				"%s has multiple instrumentation names: got %d distinct values", metricName, len(names))
		})
	}
}

func TestNodeExporterExpectedLabels(t *testing.T) {
	for _, md := range nodeExporterMetrics {
		if len(md.ExpectedLabels) == 0 {
			continue
		}
		t.Run(md.Name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), md.Name)
			requireNoError(t, err, "querying %s", md.Name)
			requireNonEmpty(t, results, "%s not available", md.Name)
			for _, r := range results {
				for _, label := range md.ExpectedLabels {
					_, ok := r.Labels.Datapoint[label]
					requireTrue(t, ok, "%s missing expected label '%s' (node: %s, host.type: %s)",
						md.Name, label,
						r.Labels.Resource["k8s.node.name"],
						r.Labels.Resource["host.type"])
				}
			}
		})
	}
}

func TestNodeExporterNoPodLabels(t *testing.T) {
	for _, metricName := range nodeExporterMetricNames {
		t.Run(metricName+"/no_container_name", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				_, has := r.Labels.Resource["k8s.container.name"]
				requireTrue(t, !has,
					"%s should not have k8s.container.name but got: %s (node: %s)",
					metricName, r.Labels.Resource["k8s.container.name"], r.Labels.Resource["k8s.node.name"])
			}
		})
		t.Run(metricName+"/no_pod_name", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				_, has := r.Labels.Resource["k8s.pod.name"]
				requireTrue(t, !has,
					"%s should not have k8s.pod.name but got: %s (node: %s)",
					metricName, r.Labels.Resource["k8s.pod.name"], r.Labels.Resource["k8s.node.name"])
			}
		})
		t.Run(metricName+"/no_namespace", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				_, has := r.Labels.Resource["k8s.namespace.name"]
				requireTrue(t, !has,
					"%s should not have k8s.namespace.name but got: %s (node: %s)",
					metricName, r.Labels.Resource["k8s.namespace.name"], r.Labels.Resource["k8s.node.name"])
			}
		})
	}
}

func TestNodeExporterNoWorkloadLabels(t *testing.T) {
	for _, metricName := range nodeExporterMetricNames {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				_, hasName := r.Labels.Resource["k8s.workload.name"]
				requireTrue(t, !hasName,
					"%s should not have k8s.workload.name but got: %s (node: %s)",
					metricName, r.Labels.Resource["k8s.workload.name"], r.Labels.Resource["k8s.node.name"])
				_, hasType := r.Labels.Resource["k8s.workload.type"]
				requireTrue(t, !hasType,
					"%s should not have k8s.workload.type but got: %s (node: %s)",
					metricName, r.Labels.Resource["k8s.workload.type"], r.Labels.Resource["k8s.node.name"])
			}
		})
	}
}

func TestNodeExporterNodeGroupCoverage(t *testing.T) {
	for _, ng := range clusterNodeGroups {
		t.Run(ng.Description+"/"+ng.InstanceType, func(t *testing.T) {
			promql := fmt.Sprintf(
				`node_load1{"@resource.k8s.cluster.name"="%s","@resource.host.type"="%s"}`,
				otelmetrics.EscapePromQLValue(cfg.ClusterName), ng.InstanceType)
			results, err := client.Query(context.Background(), promql)
			requireNoError(t, err, "querying node_load1 on %s", ng.Description)
			requireTrue(t, len(results) > 0,
				"node_exporter missing from %s nodes (%s) — DaemonSet not scheduling?",
				ng.Description, ng.InstanceType)
		})
	}
}

func TestNodeExporterHasRawNodeName(t *testing.T) {
	for _, metricName := range nodeExporterMetricNames {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				val, has := r.Labels.Datapoint["node_name"]
				requireTrue(t, has,
					"%s should have datapoint 'node_name' (raw label preserved)", metricName)
				requireTrue(t, val != "",
					"%s has empty datapoint 'node_name'", metricName)
			}
		})
	}
}
