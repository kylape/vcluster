package translate

import (
	"slices"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

func schedulingGateTestTranslator(t *testing.T) (*translator, *synccontext.SyncContext) {
	t.Helper()
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	images, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)
	tr := &translator{
		vClient: vClient, pClient: pClient, imageTranslator: images,
		eventRecorder: events.NewFakeRecorder(10), log: loghelper.New("scheduling-gates-test"),
	}
	ctx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient).ToSyncContext("test")
	assert.NilError(t, vClient.Create(ctx.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testns"}}))
	return tr, ctx
}

func TestSchedulingGateOwnershipAtCreation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		gates      []corev1.PodSchedulingGate
		wantRecord string
	}{
		{name: "guest gates", gates: []corev1.PodSchedulingGate{{Name: "guest.example/ready"}}, wantRecord: "guest.example/ready"},
		{name: "forged bookkeeping without gates"},
		{name: "Kueue gates cannot be claimed", gates: []corev1.PodSchedulingGate{{Name: "kueue.x-k8s.io/admission"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr, ctx := schedulingGateTestTranslator(t)
			guest := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns", Annotations: map[string]string{GuestSchedulingGatesAnnotation: "host.example/protected"}},
				Spec:       corev1.PodSpec{SchedulingGates: tc.gates, Containers: []corev1.Container{{Name: "c", Image: "nginx"}}},
			}
			host, err := tr.Translate(ctx, guest, nil, "", "")
			assert.NilError(t, err)
			assert.Equal(t, host.Annotations[GuestSchedulingGatesAnnotation], tc.wantRecord)
			assert.DeepEqual(t, host.Spec.SchedulingGates, tc.gates)
			assert.Equal(t, guest.Annotations[GuestSchedulingGatesAnnotation], "host.example/protected")
		})
	}
}

func TestSchedulingGateSyncPreservesHostAdmission(t *testing.T) {
	guestGate := corev1.PodSchedulingGate{Name: "guest.example/ready"}
	hostGate := corev1.PodSchedulingGate{Name: "host.example/protected"}
	admission := corev1.PodSchedulingGate{Name: "kueue.x-k8s.io/admission"}
	topology := corev1.PodSchedulingGate{Name: "kueue.x-k8s.io/topology"}
	for _, tc := range []struct {
		name                        string
		hostGates, guestGates, want []corev1.PodSchedulingGate
		record                      string
	}{
		{name: "host webhook gates with empty guest", hostGates: []corev1.PodSchedulingGate{admission, topology, hostGate}, want: []corev1.PodSchedulingGate{admission, topology, hostGate}},
		{name: "guest removes its gate", hostGates: []corev1.PodSchedulingGate{guestGate, admission, topology, hostGate}, record: guestGate.Name, want: []corev1.PodSchedulingGate{admission, topology, hostGate}},
		{name: "guest retains its gate", hostGates: []corev1.PodSchedulingGate{guestGate, admission}, guestGates: []corev1.PodSchedulingGate{guestGate}, record: guestGate.Name, want: []corev1.PodSchedulingGate{guestGate, admission}},
		{name: "host removal is never resurrected", guestGates: []corev1.PodSchedulingGate{guestGate}, record: guestGate.Name},
		{name: "new guest gates are never added after create", hostGates: []corev1.PodSchedulingGate{admission}, guestGates: []corev1.PodSchedulingGate{guestGate}, record: "other.example/gate", want: []corev1.PodSchedulingGate{admission}},
		{name: "legacy Pod ownership is never guessed", hostGates: []corev1.PodSchedulingGate{guestGate, hostGate}, want: []corev1.PodSchedulingGate{guestGate, hostGate}},
		{name: "Kueue gates protected even in a bad record", hostGates: []corev1.PodSchedulingGate{admission}, record: admission.Name, want: []corev1.PodSchedulingGate{admission}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr, ctx := schedulingGateTestTranslator(t)
			host := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"}, Spec: corev1.PodSpec{SchedulingGates: slices.Clone(tc.hostGates)}}
			if tc.record != "" {
				host.Annotations = map[string]string{GuestSchedulingGatesAnnotation: tc.record}
			}
			guest := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"}, Spec: corev1.PodSpec{SchedulingGates: slices.Clone(tc.guestGates)}}
			// Identical old/current snapshots simulate cold reconciliation after restart.
			for range 3 {
				event := synccontext.NewSyncEventWithOld(host.DeepCopy(), host, guest.DeepCopy(), guest)
				assert.NilError(t, tr.Diff(ctx, event))
				assert.Assert(t, cmp.DeepEqual(host.Spec.SchedulingGates, tc.want))
				assert.DeepEqual(t, guest.Spec.SchedulingGates, tc.guestGates)
			}
		})
	}
}

func TestGuestCannotRewriteSchedulingGateOwnership(t *testing.T) {
	tr, ctx := schedulingGateTestTranslator(t)
	host := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test", Annotations: map[string]string{GuestSchedulingGatesAnnotation: "guest.example/ready"}},
		Spec:       corev1.PodSpec{SchedulingGates: []corev1.PodSchedulingGate{{Name: "host.example/protected"}}},
	}
	guest := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"}}
	oldGuest := guest.DeepCopy()
	guest.Annotations = map[string]string{GuestSchedulingGatesAnnotation: "host.example/protected", "user.example/note": "changed"}
	assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(host.DeepCopy(), host, oldGuest, guest)))
	assert.Equal(t, host.Annotations[GuestSchedulingGatesAnnotation], "guest.example/ready")
	assert.Equal(t, host.Annotations["user.example/note"], "changed")
	assert.Equal(t, len(host.Spec.SchedulingGates), 1)
	// Guest deletion of the bookkeeping must also leave the host record intact.
	oldGuest = guest.DeepCopy()
	delete(guest.Annotations, GuestSchedulingGatesAnnotation)
	assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(host.DeepCopy(), host, oldGuest, guest)))
	assert.Equal(t, host.Annotations[GuestSchedulingGatesAnnotation], "guest.example/ready")
}
