# Infra vCluster provider build

This repository contains the patched vCluster source, Helm chart, and Tekton
definitions used by the Infra-backed sandbox PoC.

## Build outputs

The Tekton pipeline under `tekton/base/` builds the vCluster image and publishes
the matching Helm chart as an OCI artifact. PipelineRun manifests under
`tekton/runs/` select the source revision, image tag, chart version, and registry
credentials.

The Infra provider consumes both outputs. The image tag and chart version must
be treated as one release because the chart enables the CSI synchronization and
RBAC behavior implemented by the image. Do not update only one of them in the
Infra workflow.

## Runtime handoff

The Infra repository's provider runner pulls the chart from the OCI registry and
executes Helm in the target host namespace. The first target is the shared
`infra-vclusters-pirate` namespace on Pirate. Each Helm release must have a
unique short name; the chart templates use the release name to scope Services,
StatefulSets, Secrets, service accounts, Roles, bindings, Routes, and PVCs.

The runner creates the vCluster-generated RBAC. Kubernetes RBAC escalation rules
mean the runner must already possess the read permissions it asks the generated
vCluster ClusterRole to grant, including CSIStorageCapacity discovery. The
runner's target-namespace Role and cluster-scoped ClusterRole are maintained in
the Infra repository.

## Validation handoff

After publishing an image/chart pair:

1. Deploy or update the Infra workflow configuration with both immutable
   versions.
2. Create a Pirate sandbox through the `vcluster-on-pirate` flavor.
3. Confirm the vCluster service account can list CSIStorageCapacity resources.
4. Run a `WaitForFirstConsumer` PVC workload and verify host scheduling.
5. Inspect CSI syncer registration, cache events, and reconciliation logs.
6. Destroy the sandbox and verify release-specific resources are removed.

The long-term security direction is a vCluster operator installed on each host
cluster. Until that exists, keep the provider credential scoped to the PoC
namespace and avoid storing returned kubeconfig contents in source control.
