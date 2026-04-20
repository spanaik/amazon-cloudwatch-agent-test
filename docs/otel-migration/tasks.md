# OTEL Test Migration — Implementation Tasks

## Context

Port the OTEL integration tests into `amazon-cloudwatch-agent-test/` (the shared agent test
framework). Break the 13-node long-running cluster into 7 ephemeral clusters.

**Source Quip**: https://quip-amazon.com/Kg3yAcj4JPr5/MetricsV2-Testing-Coverage-Gap-Analysis-Integration-Plan
**Decision**: Agreed 15-Apr-2026 to do the recommended proposal.

### Pattern: Follow Existing Conventions Exactly

The existing test repo has a well-established pattern. We follow it — no new abstractions.

**Terraform**: Flat module per test type at `terraform/eks/daemon/{test}/main.tf`. Uses
`module "common"` (testing_id, image vars) and `module "basic_components"` (VPC, subnets, IAM).
Everything in one file — cluster, node groups, IAM roles, Helm chart, image patch, workloads, validator.

**EFA exception**: `terraform/eks/daemon/efa/` uses `terraform-aws-modules/eks/aws` v21 with
custom VPC instead of `basic_components`. Multi-EFA needs self-managed node groups.

**Tests**: Go test files under `test/{test_name}/` invoked via `null_resource.validator` with
`go test` + flags (`-eksClusterName`, `-computeType=EKS`, `-eksDeploymentStrategy=DAEMON`).

**Variables**: Standard set — `region`, `test_dir`, `cwagent_image_repo`, `cwagent_image_tag`,
`helm_chart_branch`, `k8s_version`, `ami_type`, `instance_type`.

**Generator**: Entry in `generator/test_case_generator.go` under `testTypeToTestConfig`.

### Code Quality Fixes During Port

**CRITICAL — Coverage Verification Rule**: Every task that ports tests MUST compare against
`docs/otel-migration/coverage-snapshot.md`. The snapshot lists every test function and which
cluster it must run on. Definition of done for each cluster task:
1. Count migrated test functions matches the snapshot's expected count for that cluster
2. Every test function name from the snapshot is present in the migrated code
3. Common tests (labels_common, host_validation) are importable by all cluster test packages

| Hardcoded Value | Fix |
|----------------|-----|
| `"monitoring"` SigV4 service | Config field `SigningService`, default `"monitoring"` |
| `"granite.amazonaws.com"` endpoint | **Remove `ZEUS_ENDPOINT` entirely.** Always derive from region: `https://monitoring.<region>.amazonaws.com`. No override, no config, no way to get it wrong. |
| `"metricsv2-testing"` cluster name | Require `CLUSTER_NAME` env var, no default |
| `"us-east-1"` region | Require `AWS_REGION` env var, no default |
| Hardcoded instance types in lib | Move to per-test `setup_test.go` |
| `MetricsV2Client` naming | Rename to `OtelMetricsClient` |
| Swallowed K8s client error | Log or fail, don't ignore |
| No K8s API call timeouts | Add 30s context timeout |
| Duplicated PromQL escaping | Single `EscapePromQL()` in shared lib |

---

## Task 0: Shared OTEL Metrics Client Library ✅

**Status**: Complete — branch `otel/shared-lib`, commit `e60b1b2`
**Files**: `util/otelmetrics/` (client.go, config.go, models.go, query_cache.go, source_registry.go, models_test.go)

Port the Zeus PromQL client into `util/otelmetrics/`. This is the only new shared code —
everything else reuses existing repo patterns.

Replaces PR 669's simpler `util/awsservice/otlpmetricsquery.go` as the canonical Zeus client.
PR 669's code stays — it still works for EC2 OTLP tests.

---

## Task 1: Standard Cluster Terraform ✅

**Status**: Complete — branch `otel/standard-terraform`, commit `2e6bf28`
**Files**: `terraform/eks/daemon/otel/` (main.tf, variables.tf, providers.tf)

Copied from `terraform/eks/daemon/ebs/` pattern. Creates EKS cluster + t3.medium node group,
installs Helm chart, patches agent image, deploys nginx-test + EKS addon NE/KSM.

---

## Task 2: Standard Cluster Go Tests ✅

**Status**: Complete — branch `otel/standard-tests`, commit `c65ec87`
**Files**: `test/otel/standard/` (14 files, 180 test functions, 4778 lines)

All standard cluster tests ported. GPU/Neuron/EFA host_validation tests skipped (Phase 2).
**TODO**: Terraform needs StatefulSet/Job/CronJob test workloads for KSM tests.

---

## Task 3: Test Case Generator Entry ✅

**Status**: Complete — branch `otel/test-generator`, commit `18adb48`
**Files**: `generator/test_case_generator.go` (+7 lines)

Added `eks_otel` entry with standard cluster config.

---

## Task 4: GitHub Actions Workflow ✅

**Status**: Complete — branch `otel/ci-workflow`, commit `abae0b1`
**Files**: `.github/workflows/otel-cluster-test.yml`, `.github/workflows/otel-integration-test.yml`

Reusable workflow + nightly/manual trigger. Standard cluster only for Phase 1.
**TODO**: Configure `OTEL_TEST_ROLE_ARN` secret in repo.

---

## Phase 2 Clusters (backlog — each follows the same flat Terraform pattern)

### Task 5: attr_limit — `terraform/eks/daemon/otel_attr_limit/`
3× t3.medium nodes with low/mid/high label density. Port `attribute_limit_test.go`.

### Task 6: gpu — `terraform/eks/daemon/otel_gpu/`
Copy from `terraform/eks/daemon/gpu/`. Add 1× g5.xlarge (idle) + 1× g5.12xlarge (4 GPUs).
Port `dcgm_test.go` + multi-GPU parts of `multi_device_test.go`.

### Task 7: ebs_csi — `terraform/eks/daemon/otel_ebs_csi/`
Copy from `terraform/eks/daemon/ebs/`. Port `ebs_csi_test.go`.

### Task 8: neuron — `terraform/eks/daemon/otel_neuron/`
2× inf2.xlarge. May need AZ pinning. Port `neuron_test.go`.

### Task 9: efa — `terraform/eks/daemon/otel_efa/`
Copy from `terraform/eks/daemon/efa/`. 2× c5n.9xlarge. Port `efa_test.go`.

### Task 10: multi_device — `terraform/eks/daemon/otel_multi_device/`
1× trn1.32xlarge. Self-managed node group for multi-EFA. AZ pinning. Nightly only.
Port `multi_device_test.go`.

### Task 11: K8s/IAM matrix — Phase 3
Standard cluster across {1.29–1.34} × {NodeIAM, PodIdentity}. Nightly only.

---

## Execution Order

**Phase 1** (ship first): `Task 0 → 1 → 2 → 3 → 4`
**Phase 2** (backlog, parallel): Tasks 5-10
**Phase 3** (backlog): Task 11

## Branching

| Task | Branch | From |
|------|--------|------|
| 0 | `otel/shared-lib` | `main` |
| 1 | `otel/standard-terraform` | `otel/shared-lib` |
| 2 | `otel/standard-tests` | `otel/standard-terraform` |
| 3 | `otel/test-generator` | `otel/standard-tests` |
| 4 | `otel/ci-workflow` | `otel/test-generator` |
| 5-10 | `otel/{cluster}` | `main` (after Phase 1 merges) |
| 11 | `otel/k8s-iam-matrix` | `main` |

PR title: `[OTEL Migration] Task N: <desc>`

## Effort

| Phase | Tasks | Status | Output |
|-------|-------|--------|--------|
| 1 | 0-4 | ✅ Complete | 5 branches, 25 files, 6200+ lines |
| 2 | 5-10 | Backlog | |
| 3 | 11 | Backlog | |
