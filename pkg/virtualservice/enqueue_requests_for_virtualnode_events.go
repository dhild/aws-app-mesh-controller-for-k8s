package virtualservice

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForVirtualNodeEvents(referencesIndexer references.ObjectReferenceIndexer, log logr.Logger) *enqueueRequestsForVirtualNodeEvents {
	return &enqueueRequestsForVirtualNodeEvents{
		referencesIndexer: referencesIndexer,
		log:               log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForVirtualNodeEvents)(nil)

type enqueueRequestsForVirtualNodeEvents struct {
	referencesIndexer references.ObjectReferenceIndexer
	log               logr.Logger
}

// Create is called in response to an create event
func (h *enqueueRequestsForVirtualNodeEvents) Create(_ context.Context, e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Update is called in response to an update event
func (h *enqueueRequestsForVirtualNodeEvents) Update(ctx context.Context, e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	// VirtualService reconcile depends on virtualNode is active or not.
	// so we only need to trigger VirtualService reconcile if virtualNode's active status changed.
	vnOld := e.ObjectOld.(*appmesh.VirtualNode)
	vnNew := e.ObjectNew.(*appmesh.VirtualNode)

	if virtualnode.IsVirtualNodeActive(vnOld) != virtualnode.IsVirtualNodeActive(vnNew) {
		h.enqueueVirtualServicesForVirtualNode(ctx, queue, vnNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForVirtualNodeEvents) Delete(_ context.Context, e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForVirtualNodeEvents) Generic(_ context.Context, e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForVirtualNodeEvents) enqueueVirtualServicesForVirtualNode(ctx context.Context, queue workqueue.RateLimitingInterface, vn *appmesh.VirtualNode) {
	vsList := &appmesh.VirtualServiceList{}
	if err := h.referencesIndexer.Fetch(ctx, vsList, ReferenceKindVirtualNode, k8s.NamespacedName(vn)); err != nil {
		h.log.Error(err, "failed to enqueue virtualServices for virtualNode events",
			"virtualNode", k8s.NamespacedName(vn))
		return
	}
	for _, vs := range vsList.Items {
		queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vs)})
	}
}
