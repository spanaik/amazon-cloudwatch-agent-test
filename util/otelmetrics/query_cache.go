// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package otelmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// DefaultMaxConcurrency is the default number of concurrent in-flight queries.
const DefaultMaxConcurrency = 1

// promqlEscaper is the shared escaper for PromQL label values.
var promqlEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func promqlMetricSelector(metricName string) string {
	if strings.Contains(metricName, ".") {
		return fmt.Sprintf(`{"__name__"="%s",`, metricName)
	}
	return metricName + "{"
}

type cacheEntry struct {
	results []MetricResult
	err     error
}

// QueryCache provides session-scoped caching of PromQL queries.
// Each unique metric name is queried exactly once; subsequent calls return cached data.
type QueryCache struct {
	mu         sync.RWMutex
	filtered   map[string]cacheEntry
	unfiltered map[string]cacheEntry
	client     *OtelMetricsClient
	cluster    string
	hostTypes  []string
	registry   *SourceRegistry
	sem        chan struct{}
}

// QueryCacheOption configures optional QueryCache behavior.
type QueryCacheOption func(*QueryCache)

func WithHostTypes(hostTypes []string) QueryCacheOption {
	return func(qc *QueryCache) { qc.hostTypes = hostTypes }
}

func WithSourceRegistry(registry *SourceRegistry) QueryCacheOption {
	return func(qc *QueryCache) { qc.registry = registry }
}

func WithMaxConcurrency(n int) QueryCacheOption {
	return func(qc *QueryCache) { qc.sem = make(chan struct{}, n) }
}

func NewQueryCache(client *OtelMetricsClient, clusterName string, opts ...QueryCacheOption) *QueryCache {
	qc := &QueryCache{
		filtered:   make(map[string]cacheEntry),
		unfiltered: make(map[string]cacheEntry),
		client:     client,
		cluster:    clusterName,
	}
	for _, opt := range opts {
		opt(qc)
	}
	if qc.sem == nil {
		qc.sem = make(chan struct{}, DefaultMaxConcurrency)
	}
	return qc
}

// Get returns cached results for a metric filtered by cluster name.
func (qc *QueryCache) Get(ctx context.Context, metricName string) ([]MetricResult, error) {
	qc.mu.RLock()
	if entry, ok := qc.filtered[metricName]; ok {
		qc.mu.RUnlock()
		return entry.results, entry.err
	}
	qc.mu.RUnlock()

	qc.mu.Lock()
	defer qc.mu.Unlock()

	if entry, ok := qc.filtered[metricName]; ok {
		return entry.results, entry.err
	}

	escaped := promqlEscaper.Replace(qc.cluster)
	sel := promqlMetricSelector(metricName)

	var results []MetricResult
	var firstErr error

	var targetHosts []string
	clusterScopedOnly := false

	if qc.registry != nil {
		if qc.registry.IsClusterScoped(metricName) {
			clusterScopedOnly = true
		} else {
			targetHosts = qc.registry.HostTypesFor(metricName)
		}
	} else if len(qc.hostTypes) > 0 {
		targetHosts = qc.hostTypes
	}

	if clusterScopedOnly {
		promql := fmt.Sprintf(`%s"@resource.k8s.cluster.name"="%s","@resource.host.type"=""}`, sel, escaped)
		r, err := qc.client.Query(ctx, promql)
		if err != nil {
			firstErr = err
		} else {
			results = append(results, r...)
		}
	} else if len(targetHosts) > 0 {
		type queryResult struct {
			results []MetricResult
			err     error
		}
		ch := make(chan queryResult, len(targetHosts))
		var wg sync.WaitGroup

		for _, ht := range targetHosts {
			wg.Add(1)
			go func(hostType string) {
				defer wg.Done()
				select {
				case qc.sem <- struct{}{}:
					defer func() { <-qc.sem }()
				case <-ctx.Done():
					ch <- queryResult{err: ctx.Err()}
					return
				}
				promql := fmt.Sprintf(`%s"@resource.k8s.cluster.name"="%s","@resource.host.type"="%s"}`, sel, escaped, hostType)
				r, err := qc.client.Query(ctx, promql)
				ch <- queryResult{results: r, err: err}
			}(ht)
		}

		go func() { wg.Wait(); close(ch) }()

		for qr := range ch {
			if qr.err != nil {
				if firstErr == nil {
					firstErr = qr.err
				}
				continue
			}
			results = append(results, qr.results...)
		}
	} else {
		promql := fmt.Sprintf(`%s"@resource.k8s.cluster.name"="%s"}`, sel, escaped)
		r, err := qc.client.Query(ctx, promql)
		results = r
		firstErr = err
	}

	if len(results) == 0 && firstErr != nil {
		slog.Debug("query failed", "metric", metricName, "error", firstErr)
		qc.filtered[metricName] = cacheEntry{err: firstErr}
		return nil, firstErr
	}
	qc.filtered[metricName] = cacheEntry{results: results}
	return results, nil
}

// GetWithFilter returns results with additional PromQL label filters. Not cached.
func (qc *QueryCache) GetWithFilter(ctx context.Context, metricName string, extraFilters map[string]string) ([]MetricResult, error) {
	escaped := promqlEscaper.Replace(qc.cluster)
	sel := promqlMetricSelector(metricName)

	filters := fmt.Sprintf(`"@resource.k8s.cluster.name"="%s"`, escaped)
	for key, value := range extraFilters {
		escapedVal := promqlEscaper.Replace(value)
		switch {
		case strings.HasPrefix(key, "~@resource."):
			filters += fmt.Sprintf(`,"%s"=~"%s"`, strings.TrimPrefix(key, "~"), escapedVal)
		case strings.HasPrefix(key, "@resource."):
			filters += fmt.Sprintf(`,"%s"="%s"`, key, escapedVal)
		case strings.HasPrefix(key, "~"):
			filters += fmt.Sprintf(`,%s=~"%s"`, strings.TrimPrefix(key, "~"), escapedVal)
		default:
			filters += fmt.Sprintf(`,%s="%s"`, key, escapedVal)
		}
	}

	return qc.client.Query(ctx, sel+filters+"}")
}

// GetUnfiltered returns results without cluster filtering. Cached separately.
func (qc *QueryCache) GetUnfiltered(ctx context.Context, metricName string) ([]MetricResult, error) {
	qc.mu.RLock()
	if entry, ok := qc.unfiltered[metricName]; ok {
		qc.mu.RUnlock()
		return entry.results, entry.err
	}
	qc.mu.RUnlock()

	qc.mu.Lock()
	defer qc.mu.Unlock()

	if entry, ok := qc.unfiltered[metricName]; ok {
		return entry.results, entry.err
	}

	results, err := qc.client.Query(ctx, metricName)
	qc.unfiltered[metricName] = cacheEntry{results: results, err: err}
	return results, err
}
