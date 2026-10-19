package translate

import (
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// GuestSchedulingGatesAnnotation is host-only syncer bookkeeping. Gate names
// cannot contain commas, so an immutable comma-separated list is sufficient.
const GuestSchedulingGatesAnnotation = "vcluster.loft.sh/guest-scheduling-gates"

func hostManagedSchedulingGate(name string) bool {
	// A guest must not claim ownership of Kueue gates by declaring the same name.
	return strings.HasPrefix(name, "kueue.x-k8s.io/")
}

func recordGuestSchedulingGates(host, guest *corev1.Pod) {
	var names []string
	for _, gate := range guest.Spec.SchedulingGates {
		if !hostManagedSchedulingGate(gate.Name) {
			names = append(names, gate.Name)
		}
	}
	// Always replace guest-supplied bookkeeping during host Pod creation.
	delete(host.Annotations, GuestSchedulingGatesAnnotation)
	if len(names) != 0 {
		if host.Annotations == nil {
			host.Annotations = map[string]string{}
		}
		host.Annotations[GuestSchedulingGatesAnnotation] = strings.Join(names, ",")
	}
}

func syncSchedulingGates(host, guest *corev1.Pod) {
	recorded, tracked := host.Annotations[GuestSchedulingGatesAnnotation]
	if !tracked {
		// Older Pods have no durable ownership record. Never infer ownership from
		// a mutable guest object: preserving their host gates is the safe fallback.
		return
	}
	owned := strings.Split(recorded, ",")
	host.Spec.SchedulingGates = slices.DeleteFunc(slices.Clone(host.Spec.SchedulingGates), func(gate corev1.PodSchedulingGate) bool {
		if hostManagedSchedulingGate(gate.Name) || !slices.Contains(owned, gate.Name) {
			return false
		}
		return !slices.ContainsFunc(guest.Spec.SchedulingGates, func(current corev1.PodSchedulingGate) bool {
			return current.Name == gate.Name
		})
	})
	// Work only from the live host list. Gates removed by host controllers must
	// never be re-added from stale guest state, nor reverse-synced into a guest
	// Pod that already exists (Kubernetes permits only gate removal after create).
}
