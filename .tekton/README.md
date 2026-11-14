Tekton CI definitions for this repository, rendered by the polling CI
service (gitlab.cee.redhat.com/klape/build). The poller:

* creates a PipelineRun from `build.yaml` for every push to the default
  branch (push trigger only — the fork does not gate PRs)
* substitutes the `repo-url`, `revision`, `image-name` and `push-latest`
  params and injects the shared RWX source workspace
* publishes images to the shared quay org as `<repo>:<sha>` (and `:latest`
  on default-branch pushes) — free image publishing for the pinned fork
* reports results as `ci/build` plus per-task commit statuses

See the build repo README for the full design and onboarding docs.
