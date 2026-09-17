package csistoragecapacities

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := CreateCSIStorageCapacitiesMapper(ctx)
	if err != nil {
		return nil, err
	}

	return &csistoragecapacitySyncer{
		Mapper: mapper,

		storageClassSyncEnabled:     ctx.Config.Sync.ToHost.StorageClasses.Enabled,
		hostStorageClassSyncEnabled: ctx.Config.Sync.FromHost.StorageClasses.Enabled == "true",
		hostClient:                  ctx.HostManager.GetClient(),
	}, nil
}

type csistoragecapacitySyncer struct {
	synccontext.Mapper
	hostClient                  client.Client
	storageClassSyncEnabled     bool
	hostStorageClassSyncEnabled bool
}

var _ syncertypes.Syncer = &csistoragecapacitySyncer{}

func (s *csistoragecapacitySyncer) Name() string {
	return "csistoragecapacity"
}

func (s *csistoragecapacitySyncer) Resource() client.Object {
	return &storagev1.CSIStorageCapacity{}
}

func (s *csistoragecapacitySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *csistoragecapacitySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.CSIStorageCapacity]) (ctrl.Result, error) {
	ctx.Log.Infof("CSIStorageCapacity SyncToVirtual entered: host=%s/%s storageClass=%s", event.Host.Namespace, event.Host.Name, event.Host.StorageClassName)
	vObj, shouldSkip, err := s.translateBackwards(ctx, event.Host)
	if err != nil || shouldSkip {
		ctx.Log.Infof("CSIStorageCapacity skipped: host=%s/%s skip=%t error=%v", event.Host.Namespace, event.Host.Name, shouldSkip, err)
		return ctrl.Result{}, err
	}
	ctx.Log.Infof("CSIStorageCapacity mapped: host=%s/%s guest=%s/%s storageClass=%s", event.Host.Namespace, event.Host.Name, vObj.Namespace, vObj.Name, vObj.StorageClassName)

	// Apply pro patches
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.CSIStorageCapacities.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create CSIStorageCapacity %s, because it does not exist in virtual cluster", vObj.Name)
	err = ctx.VirtualClient.Create(ctx, vObj)
	ctx.Log.Infof("CSIStorageCapacity guest create: guest=%s/%s error=%v", vObj.Namespace, vObj.Name, err)
	return ctrl.Result{}, err
}

func (s *csistoragecapacitySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.CSIStorageCapacity]) (_ ctrl.Result, retErr error) {
	ctx.Log.Infof("CSIStorageCapacity Sync entered: host=%s/%s guest=%s/%s", event.Host.Namespace, event.Host.Name, event.Virtual.Namespace, event.Virtual.Name)
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.CSIStorageCapacities.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	shouldSkip := false

	defer func() {
		if shouldSkip {
			// the virtual object was deleted so we don't patch
			return
		}
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// check if there is a change
	shouldSkip, err = s.translateUpdateBackwards(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if shouldSkip {
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
	}

	return ctrl.Result{}, nil
}

func (s *csistoragecapacitySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.CSIStorageCapacity]) (ctrl.Result, error) {
	ctx.Log.Infof("CSIStorageCapacity SyncToHost entered: guest=%s/%s", event.Virtual.Namespace, event.Virtual.Name)
	ctx.Log.Infof("delete virtual CSIStorageCapacity %s, because physical object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *csistoragecapacitySyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	// the default cache is configured to look at only the target namespaces, create an event source from
	// a cache that watches all namespaces
	allNSCache, err := cache.New(ctx.HostManager.GetConfig(), cache.Options{
		Scheme: ctx.HostManager.GetScheme(),
		Mapper: ctx.HostManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create allNSCache: %w", err)
	}
	if _, err := allNSCache.GetInformer(ctx, s.Resource()); err != nil {
		return nil, fmt.Errorf("failed to create CSIStorageCapacity informer: %w", err)
	}
	if err := ctx.HostManager.Add(allNSCache); err != nil {
		return nil, fmt.Errorf("failed to add allNSCache to physical manager: %w", err)
	}
	hostCapacities := &storagev1.CSIStorageCapacityList{}
	if err := ctx.HostManager.GetAPIReader().List(ctx, hostCapacities); err != nil {
		klog.Warningf("CSI capacity direct host list failed after cache registration: %v", err)
	} else {
		klog.Infof("CSI capacity direct host list after cache registration: count=%d", len(hostCapacities.Items))
	}
	klog.Infof("CSI capacity auxiliary cache added to host manager")

	syncContext := ctx.ToSyncContext("csi storage capacity syncer")
	return builder.WatchesRawSource(source.Kind(allNSCache, s.Resource(), &handler.Funcs{
		CreateFunc: func(_ context.Context, ce event.TypedCreateEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ce.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
		UpdateFunc: func(_ context.Context, ue event.TypedUpdateEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ue.ObjectNew
			s.enqueuePhysical(syncContext, obj, rli)
		},
		DeleteFunc: func(_ context.Context, de event.TypedDeleteEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := de.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
		GenericFunc: func(_ context.Context, ge event.TypedGenericEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ge.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
	})), nil
}

func (s *csistoragecapacitySyncer) enqueuePhysical(ctx *synccontext.SyncContext, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	if obj == nil {
		return
	}

	name := s.HostToVirtual(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	ctx.Log.Infof("CSIStorageCapacity host cache event: host=%s/%s guest=%s/%s", obj.GetNamespace(), obj.GetName(), name.Namespace, name.Name)
	if name.Name != "" && name.Namespace != "" {
		q.Add(reconcile.Request{NamespacedName: name})
	}
}
