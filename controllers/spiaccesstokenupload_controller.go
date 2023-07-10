package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	api "github.com/redhat-appstudio/service-provider-integration-operator/api/v1beta1"
	opconfig "github.com/redhat-appstudio/service-provider-integration-operator/pkg/config"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/serviceprovider"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/spi-shared/tokenstorage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/finalizer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// SPIAccessTokenUploadReconciler reconciles a SPIAccessTokenUpload object
type SPIAccessTokenUploadReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	TokenStorage           tokenstorage.TokenStorage
	Configuration          *opconfig.OperatorConfiguration
	ServiceProviderFactory serviceprovider.Factory
	finalizers             finalizer.Finalizers
}

//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=spiaccesstokenuploads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=spiaccesstokenuploads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=spiaccesstokenuploads/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *SPIAccessTokenUploadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.finalizers = finalizer.NewFinalizers()

	if err := r.finalizers.Register(tokenStorageFinalizerName, &tokenStorageFinalizer{storage: r.TokenStorage}); err != nil {
		return fmt.Errorf("failed to register the token storage finalizer: %w", err)
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&api.SPIAccessTokenUpload{}).
		//Watches(&source.Kind{Type: &api.SPIAccessTokenDataUpdate{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
		//	return requestForDataUpdateOwner(object, "SPIAccessToken", true)
		//})).
		Complete(r)

	if err != nil {
		err = fmt.Errorf("failed to build the controller manager: %w", err)
	}

	return err
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SPIAccessTokenUploadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("starting reconciliation")
	defer logs.TimeTrackWithLazyLogger(func() logr.Logger { return lg }, time.Now(), "Reconcile SPIAccessTokenUpload")

	at := api.SPIAccessTokenUpload{}

	if err := r.Get(ctx, req.NamespacedName, &at); err != nil {
		if errors.IsNotFound(err) {
			lg.V(logs.DebugLevel).Info("token upload already gone from the cluster. skipping reconciliation")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to load the token from the cluster: %w", err)
	}
	lg = lg.WithValues("phase_at_reconcile_start", at.Status.Phase)
	log.IntoContext(ctx, lg)
	finalizationResult, err := r.finalizers.Finalize(ctx, &at)
	if err != nil {
		// if the finalization fails, the finalizer stays in place, and so we don't want any repeated attempts until
		// we get another reconciliation due to cluster state change
		return ctrl.Result{Requeue: false}, fmt.Errorf("failed to finalize: %w", err)
	}
	if finalizationResult.Updated {
		if err = r.Client.Update(ctx, &at); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update based on finalization result: %w", err)
		}
	}
	if finalizationResult.StatusUpdated {
		if err = r.Client.Status().Update(ctx, &at); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update the status based on finalization result: %w", err)
		}
	}

	if at.DeletionTimestamp != nil {
		lg.V(logs.DebugLevel).Info("token being deleted, no other changes required after completed finalization")
		return ctrl.Result{}, nil
	}
	lg.Info("TODO<--------------")
	return ctrl.Result{}, nil
}
