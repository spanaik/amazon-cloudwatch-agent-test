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

const scopeKubeletstats = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver"

var kubeletstatsMetricNamesList = metricNames(kubeletstatsMetrics)

func TestKubeletstatsMetricExistence(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
		})
	}
}

func TestKubeletstatsInstrumentationSource(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				name, ok := r.Labels.Instrumentation["@name"]
				requireTrue(t, ok, "%s missing @instrumentation.@name", metricName)
				requireEqual(t, name, scopeKubeletstats,
					"%s instrumentation name", metricName)
			}
		})
	}
}

func TestKubeletstatsClusterIdentity(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				requireEqual(t, r.Labels.Resource["k8s.cluster.name"], cfg.ClusterName,
					"%s cluster name", metricName)
			}
		})
	}
}

func TestKubeletstatsNodeName(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				node, ok := r.Labels.Resource["k8s.node.name"]
				requireTrue(t, ok, "%s missing @resource.k8s.node.name", metricName)
				requireTrue(t, node != "", "%s has empty @resource.k8s.node.name", metricName)
			}
		})
	}
}

func TestKubeletstatsPodName(t *testing.T) {
	var podAndContainerMetrics []string
	podAndContainerMetrics = append(podAndContainerMetrics, metricNames(kubeletstatsPodMetrics)...)
	podAndContainerMetrics = append(podAndContainerMetrics, metricNames(kubeletstatsContainerMetrics)...)
	for _, metricName := range podAndContainerMetrics {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				pod, ok := r.Labels.Resource["k8s.pod.name"]
				requireTrue(t, ok, "%s missing @resource.k8s.pod.name", metricName)
				requireTrue(t, pod != "", "%s has empty @resource.k8s.pod.name", metricName)
			}
		})
	}
}

func TestKubeletstatsNamespace(t *testing.T) {
	var podAndContainerMetrics []string
	podAndContainerMetrics = append(podAndContainerMetrics, metricNames(kubeletstatsPodMetrics)...)
	podAndContainerMetrics = append(podAndContainerMetrics, metricNames(kubeletstatsContainerMetrics)...)
	for _, metricName := range podAndContainerMetrics {
		t.Run(metricName, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				ns, ok := r.Labels.Resource["k8s.namespace.name"]
				requireTrue(t, ok, "%s missing @resource.k8s.namespace.name", metricName)
				requireTrue(t, ns != "", "%s has empty @resource.k8s.namespace.name", metricName)
			}
		})
	}
}

func TestKubeletstatsContainerName(t *testing.T) {
	for _, md := range kubeletstatsContainerMetrics {
		t.Run(md.Name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), md.Name)
			requireNoError(t, err, "querying %s", md.Name)
			requireNonEmpty(t, results, "%s not available", md.Name)
			hasContainer := 0
			for _, r := range results {
				if cn, ok := r.Labels.Resource["k8s.container.name"]; ok {
					requireTrue(t, cn != "", "%s has empty k8s.container.name", md.Name)
					hasContainer++
				}
			}
			requireTrue(t, hasContainer > 0,
				"%s has no results with k8s.container.name", md.Name)
		})
	}
}

func TestKubeletstatsCloudDetection(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName+"/provider", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				requireEqual(t, r.Labels.Resource["cloud.provider"], "aws",
					"%s cloud.provider", metricName)
			}
		})
		t.Run(metricName+"/region", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				region, ok := r.Labels.Resource["cloud.region"]
				requireTrue(t, ok, "%s missing @resource.cloud.region", metricName)
				requireTrue(t, region != "", "%s has empty cloud.region", metricName)
			}
		})
		t.Run(metricName+"/account_id", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				acctID, ok := r.Labels.Resource["cloud.account.id"]
				requireTrue(t, ok, "%s missing @resource.cloud.account.id", metricName)
				requireEqual(t, len(acctID), 12, "%s cloud.account.id length", metricName)
			}
		})
	}
}

func TestKubeletstatsHostDetection(t *testing.T) {
	for _, metricName := range kubeletstatsMetricNamesList {
		t.Run(metricName+"/host_id", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				hostID, ok := r.Labels.Resource["host.id"]
				requireTrue(t, ok, "%s missing @resource.host.id", metricName)
				requireTrue(t, strings.HasPrefix(hostID, "i-"),
					"%s host.id should start with 'i-', got %q", metricName, hostID)
			}
		})
		t.Run(metricName+"/host_name", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				hostName, ok := r.Labels.Resource["host.name"]
				requireTrue(t, ok, "%s missing @resource.host.name", metricName)
				requireTrue(t, hostName != "", "%s has empty host.name", metricName)
			}
		})
		t.Run(metricName+"/host_type", func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), metricName)
			requireNoError(t, err, "querying %s", metricName)
			requireNonEmpty(t, results, "%s not available", metricName)
			for _, r := range results {
				hostType, ok := r.Labels.Resource["host.type"]
				requireTrue(t, ok, "%s missing @resource.host.type", metricName)
				requireTrue(t, hostType != "", "%s has empty host.type", metricName)
			}
		})
	}
}

func TestKubeletstatsExpectedLabels(t *testing.T) {
	for _, md := range kubeletstatsMetrics {
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
					requireTrue(t, ok, "%s missing expected label '%s' (node: %s, host.type: %s, pod: %s)",
						md.Name, label,
						r.Labels.Resource["k8s.node.name"],
						r.Labels.Resource["host.type"],
						r.Labels.Resource["k8s.pod.name"])
				}
			}
		})
	}
}

func TestKubeletstatsUnitValidation(t *testing.T) {
	for _, md := range kubeletstatsMetrics {
		if md.Unit == "" {
			continue
		}
		t.Run(md.Name, func(t *testing.T) {
			results, err := queryCache.Get(context.Background(), md.Name)
			requireNoError(t, err, "querying %s", md.Name)
			requireNonEmpty(t, results, "%s not available", md.Name)
			for _, r := range results {
				unit, ok := r.Labels.Datapoint["__unit__"]
				requireTrue(t, ok, "%s missing __unit__ label", md.Name)
				requireEqual(t, unit, md.Unit, "%s unit", md.Name)
			}
		})
	}
}

func TestKubeletstatsNodeGroupCoverage(t *testing.T) {
	for _, ng := range clusterNodeGroups {
		t.Run(ng.Description+"/"+ng.InstanceType, func(t *testing.T) {
			promql := fmt.Sprintf(
				`{"__name__"="k8s.node.cpu.utilization","@resource.k8s.cluster.name"="%s","@resource.host.type"="%s"}`,
				otelmetrics.EscapePromQLValue(cfg.ClusterName), ng.InstanceType)
			results, err := client.Query(context.Background(), promql)
			requireNoError(t, err, "querying k8s.node.cpu.utilization on %s", ng.Description)
			requireTrue(t, len(results) > 0,
				"kubeletstats missing from %s nodes (%s) — kubeletstatsreceiver not scraping?",
				ng.Description, ng.InstanceType)
		})
	}
}
