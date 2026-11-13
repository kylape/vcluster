# Archived Tekton manifests

`base/` contains the persistent resources managed by Kustomize.

`runs/` contains the current one-shot `PipelineRun` manifest. Create a new run
with `kubectl create -k runs`; do not use `kubectl apply` for PipelineRuns.

`product-task/` contains the failed experiment using the Red Hat-provided
Buildah Task. It is retained for comparison but is not part of the active
Kustomize base.

`previous-runs/` contains superseded PipelineRun manifests.
