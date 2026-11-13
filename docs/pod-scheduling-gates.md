# Pod scheduling gate synchronization

Host admission controllers can add scheduling gates when vCluster creates a
physical Pod. The Pod translator preserves these gates instead of replacing
the host list with the guest Pod's list during subsequent reconciliation.

## Ownership

At host Pod creation, the syncer records guest-origin gate names in the
host-only `vcluster.loft.sh/guest-scheduling-gates` annotation. It generates the
record from the guest spec, replacing any guest-supplied value. Subsequent
annotation synchronization excludes this key in both directions.

Reconciliation starts from the live host gate list. A gate is removed only when
its name is recorded as guest-origin and it is absent from the current guest
spec. Other host gates remain untouched. Gates removed by a host controller are
never added back from stale guest state.

The `kueue.x-k8s.io/` gate namespace is always treated as host-controlled, even
when a guest originally declares a matching name. This prevents the syncer from
acting on guest requests to release Kueue admission or topology gates.

## Existing Pods and visibility

Pods created by older syncers lack a durable ownership record. The syncer
preserves their host gates rather than guessing ownership from the mutable guest
spec. Recreate such Pods to use guest-controlled gate removal under this scheme.

Host-added gates are not copied into an existing guest Pod's spec: Kubernetes
allows gate additions only during Pod creation. Host Pod status continues to
propagate into the guest, providing scheduling status without transferring gate
ownership.

## Enforcement boundary

This change prevents accidental gate removal by vCluster. It does not add a
host admission policy restricting which callers may remove scheduling gates,
nor require that every tenant workload participate in Kueue.
