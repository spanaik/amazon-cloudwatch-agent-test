//go:build integration

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package standard

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/amazon-cloudwatch-agent-test/util/otelmetrics"
)

var cadvisorMetricNamesList = metricNames(cadvisorMetrics)

func TestCadvisorInstrumentationSource(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			for _, r := range results {
				v, ok := r.Labels.Instrumentation["@name"]
				requireTrue(t, ok, "%s missing @instrumentation.@name", name)
				requireEqual(t, v, "github.com/google/cadvisor", "%s instrumentation", name)
			}
		})
	}
}

func TestCadvisorInstrumentationConsistent(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			names := make(map[string]struct{})
			for _, r := range results {
				if n, ok := r.Labels.Instrumentation["@name"]; ok {
					names[n] = struct{}{}
				}
			}
			requireEqual(t, len(names), 1, "%s has %d distinct instrumentation names", name, len(names))
		})
	}
}

func TestCadvisorPodName(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			for _, r := range results {
				pod := r.Labels.Resource["k8s.pod.name"]
				requireTrue(t, pod != "", "%s missing k8s.pod.name", name)
			}
		})
	}
}

func TestCadvisorNamespace(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			for _, r := range results {
				ns := r.Labels.Resource["k8s.namespace.name"]
				requireTrue(t, ns != "", "%s missing k8s.namespace.name", name)
			}
		})
	}
}

func TestCadvisorContainerName(t *testing.T) {
	for _, md := range cadvisorMetrics {
		if md.Scope != otelmetrics.ScopeContainer {
			continue
		}
		t.Run(md.Name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), md.Name)
			requireNoError(t, err, "querying %s", md.Name)
			requireNonEmpty(t, results, "%s not available", md.Name)
			hasContainer := 0
			for _, r := range results {
				if cn, ok := r.Labels.Resource["k8s.container.name"]; ok {
					requireTrue(t, cn != "", "%s empty k8s.container.name", md.Name)
					hasContainer++
				}
			}
			requireTrue(t, hasContainer > 0, "%s no results with k8s.container.name", md.Name)
		})
	}
}

func TestCadvisorExpectedLabels(t *testing.T) {
	for _, md := range cadvisorMetrics {
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
					requireTrue(t, ok, "%s missing expected label '%s'", md.Name, label)
				}
			}
		})
	}
}

func TestCadvisorPodLabelsStrict(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			for _, r := range results {
				requireTrue(t, r.Labels.Resource["k8s.pod.name"] != "", "%s missing pod", name)
				requireTrue(t, r.Labels.Resource["k8s.namespace.name"] != "", "%s missing ns", name)
				requireTrue(t, r.Labels.Resource["k8s.node.name"] != "", "%s missing node", name)
			}
		})
	}
}

func TestCadvisorNginxWorkloadLabels(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			promql := fmt.Sprintf(`%s{"@resource.k8s.cluster.name"="%s","@resource.k8s.pod.name"=~"nginx-test.*"}`,
				name, cfg.ClusterName)
			nginx, err := client.Query(context.Background(), promql)
			requireNoError(t, err, "querying %s for nginx-test", name)
			requireTrue(t, len(nginx) > 0, "No %s from nginx-test pods", name)
			for _, r := range nginx {
				requireEqual(t, r.Labels.Resource["k8s.workload.name"], "nginx-test", "%s workload name", name)
				requireEqual(t, r.Labels.Resource["k8s.workload.type"], "Deployment", "%s workload type", name)
			}
		})
	}
}

func TestCadvisorHasRawPromotedKeys(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			rawKeys := []string{"pod", "namespace"}
			if !strings.Contains(name, "network") {
				rawKeys = append(rawKeys, "container")
			}
			for _, key := range rawKeys {
				found := false
				for _, r := range results {
					if v, ok := r.Labels.Datapoint[key]; ok && v != "" {
						found = true
						break
					}
				}
				requireTrue(t, found, "%s missing raw '%s' at datapoint scope", name, key)
			}
		})
	}
}

func TestCadvisorNoPodSandboxMetrics(t *testing.T) {
	for _, name := range cadvisorMetricNamesList {
		t.Run(name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), name)
			requireNoError(t, err, "querying %s", name)
			requireNonEmpty(t, results, "%s not available", name)
			for _, r := range results {
				requireTrue(t, r.Labels.Resource["k8s.container.name"] != "POD", "%s has POD container", name)
				requireTrue(t, r.Labels.Datapoint["container"] != "POD", "%s has POD datapoint", name)
			}
		})
	}
}

func TestCadvisorNodeGroupCoverage(t *testing.T) {
	for _, ng := range clusterNodeGroups {
		t.Run(ng.Description+"/"+ng.InstanceType, func(t *testing.T) {
			promql := fmt.Sprintf(
				`container_memory_working_set_bytes{"@resource.k8s.cluster.name"="%s","@resource.host.type"="%s"}`,
				otelmetrics.EscapePromQLValue(cfg.ClusterName), ng.InstanceType)
			results, err := client.Query(context.Background(), promql)
			requireNoError(t, err, "querying cadvisor on %s", ng.Description)
			requireTrue(t, len(results) > 0, "cadvisor missing from %s (%s)", ng.Description, ng.InstanceType)
		})
	}
}
