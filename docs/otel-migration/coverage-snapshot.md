# OTEL Test Coverage Snapshot (Baseline)

Generated: 2026-04-16T22:17:36Z
Source: `cloudwatch-agent-testing/tests-go/integration/`
**Total: 279 test functions across 21 files**

## Purpose

This is the baseline snapshot of all OTEL integration tests before migration. Every test
listed here MUST be accounted for in the migrated test suite. Use this as the definition
of done — compare against this snapshot after each task to verify no coverage is lost.

## Baseline Run (2026-04-16)

Cluster: `metricsv2-testing` (us-east-1, account 322962300869)
Endpoint: `https://monitoring.us-east-1.amazonaws.com` (prod)
**Result: 276 PASS / 1 FAIL / 277 total**

The 1 failure is `TestKSM_PodBucket_JobPodName` — flaky due to K8s Job pod lifecycle timing.
KSM may not have the Job pod in its scrape window after the Job completes.

**IMPORTANT**: The original test client has a configurable `ZEUS_ENDPOINT` that defaults to
`granite.amazonaws.com` (preprod). This caused 233 false failures when the cluster exports to
prod. The migrated code MUST NOT have an endpoint override — always derive from region:
`https://monitoring.<region>.amazonaws.com`. This eliminates the #1 source of test misconfiguration.

## Coverage by Cluster

### Common tests (run on EVERY cluster): 41 tests

These tests validate cross-cutting concerns and MUST run on every cluster type.

**labels_common_test.go (19):**
- TestClusterIdentity, TestAWSLabels, TestInstrumentationLabels
- TestScopeNameAndVersionPerSource, TestCloudResourceDetection
- TestHostResourceDetection, TestNodeMetricLabels, TestPodMetricLabels
- TestContainerMetricLabels, TestLabelPreservation, TestMetricNamePreservation
- TestServiceLabels, TestNodeLabelEnrichment, TestWorkloadLabels
- TestMetricTypeLabels, TestMetricUnitLabels, TestCustomNodeLabels
- TestCloudResourceId, TestScrapeMetadataFiltered

**host_validation_test.go (22):**
- TestHostTypeMatchesNodeGroup, TestHostNameMatchesNodeName
- TestUniqueHostIdPerNode, TestCloudRegionMatchesConfig
- TestCloudAccountMatchesConfig, TestCloudAvailabilityZonePresent
- TestCloudProviderIsAWS, TestCloudPlatformIsEKS, TestCloudResourceIdIsEKSArn
- TestMultiGPUHostType, TestMultiNeuronHostType
- TestPodScheduledOnCorrectNode, TestHostIdMatchesProviderID
- TestNodeUidMatchesKubernetesAPI, TestContainerExistsInPodSpec
- TestPodNamespaceMatchesKubernetesAPI, TestNodeLabelsMatchKubernetesAPI
- TestDeploymentOwnerMatchesKubernetesAPI, TestStatefulSetOwnerMatchesKubernetesAPI
- TestDaemonSetOwnerMatchesKubernetesAPI, TestJobOwnerMatchesKubernetesAPI
- TestAllNodeGroupsPresent

### DaemonSet metric tests (run on standard + all hardware clusters): 30 tests

**cadvisor_test.go (11):**
- TestCadvisorInstrumentationSource, TestCadvisorInstrumentationConsistent
- TestCadvisorPodName, TestCadvisorNamespace, TestCadvisorContainerName
- TestCadvisorExpectedLabels, TestCadvisorPodLabelsStrict
- TestCadvisorNginxWorkloadLabels, TestCadvisorHasRawPromotedKeys
- TestCadvisorNoPodSandboxMetrics, TestCadvisorNodeGroupCoverage

**node_exporter_test.go (7):**
- TestNodeExporterInstrumentationSource, TestNodeExporterInstrumentationConsistent
- TestNodeExporterExpectedLabels, TestNodeExporterNoPodLabels
- TestNodeExporterNoWorkloadLabels, TestNodeExporterNodeGroupCoverage
- TestNodeExporterHasRawNodeName

**kubeletstats_test.go (12):**
- TestKubeletstatsMetricExistence, TestKubeletstatsInstrumentationSource
- TestKubeletstatsClusterIdentity, TestKubeletstatsNodeName
- TestKubeletstatsPodName, TestKubeletstatsNamespace
- TestKubeletstatsContainerName, TestKubeletstatsCloudDetection
- TestKubeletstatsHostDetection, TestKubeletstatsExpectedLabels
- TestKubeletstatsUnitValidation, TestKubeletstatsNodeGroupCoverage

### Standard cluster only: 104 tests

**kube_state_metrics_test.go (90):** All TestKSM_* tests
**control_plane_test.go (8):** All TestControlPlane*/TestAPIServer* tests
**eks_addon_test.go (6):** All TestEksAddon*/TestBundled* tests

### Dedup + Resolution (per-source, split across clusters): 8 tests

**dedup_test.go (7):**
- TestNodeExporterNoDuplicateSeries → standard
- TestCadvisorNoDuplicateSeries → standard
- TestNodeExporterNoSchemaUrlLabel → standard
- TestKubeletstatsNoDuplicateSeries → standard
- TestDCGMNoDuplicateSeries → gpu
- TestNeuronNoDuplicateSeries → neuron
- TestEFANoDuplicateSeries → efa

**resolution_test.go (1):**
- TestMetricResolution → runs per-source (node_exporter on standard, DCGM on gpu, etc.)

### GPU cluster: 17 tests

**dcgm_test.go (17):** All TestDCGM* tests

### Neuron cluster: 19 tests

**neuron_test.go (19):** All TestNeuron* tests

### EFA cluster: 18 tests

**efa_test.go (18):** All TestEFA* tests

### Multi-device cluster: 25 tests

**multi_device_test.go (25):** All TestMulti* tests

### Attribute limit cluster: 7 tests

**attribute_limit_test.go (7):** All TestPhase*/TestProtected*/TestAttribute* tests

### EBS CSI cluster: 5 tests

**ebs_csi_test.go (5):** All TestEBSCSI* tests

### Unit tests (no cluster needed): 4 tests

**k8s_helpers_unit_test.go (4):**
- TestParseInstanceIDFromProviderID, TestLookupPod
- TestParseAZFromProviderID, TestGroundTruthIndexing

## Verification Checklist

After migrating each cluster, run this check:

```bash
# Count tests in migrated cluster
grep -c '^func Test' test/otel/{cluster}/*_test.go test/otel/common/*_test.go

# Compare against this snapshot's count for that cluster
```

| Cluster | Expected Test Count | Source |
|---------|-------------------|--------|
| standard | 41 (common) + 30 (daemonset) + 104 (ksm+cp+addon) + 4 (dedup) + 1 (resolution) = **180** | labels_common + host_validation + cadvisor + node_exporter + kubeletstats + ksm + control_plane + eks_addon + dedup(4) + resolution |
| attr_limit | 41 (common) + 30 (daemonset) + 7 (attr_limit) = **78** | labels_common + host_validation + cadvisor + node_exporter + kubeletstats + attribute_limit |
| gpu | 41 (common) + 30 (daemonset) + 17 (dcgm) + 1 (dedup) = **89** | labels_common + host_validation + cadvisor + node_exporter + kubeletstats + dcgm + dedup(1) |
| neuron | 41 (common) + 30 (daemonset) + 19 (neuron) + 1 (dedup) = **91** | labels_common + host_validation + cadvisor + node_exporter + kubeletstats + neuron + dedup(1) |
| efa | 41 (common) + 30 (daemonset) + 18 (efa) + 1 (dedup) = **90** | labels_common + host_validation + cadvisor + node_exporter + kubeletstats + efa + dedup(1) |
| multi_device | 41 (common) + 25 (multi_device) = **66** | labels_common + host_validation + multi_device |
| ebs_csi | 41 (common) + 5 (ebs_csi) = **46** | labels_common + host_validation + ebs_csi |
| unit (no cluster) | **4** | k8s_helpers_unit |
| **TOTAL** | **644** (279 unique, some run on multiple clusters) | |
