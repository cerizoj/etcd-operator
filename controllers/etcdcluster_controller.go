/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	databasev1alpha1 "github.com/etcd-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var etcdClusterFinalizerName = "database.github.com/finalizer"

// EtcdClusterReconciler reconciles a EtcdCluster object
type EtcdClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=database.github.com,resources=etcdclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.github.com,resources=etcdclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.github.com,resources=etcdclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *EtcdClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	cr := &databasev1alpha1.EtcdCluster{}
	l.Info("receive request")
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		l.Error(err, "fetch etcd cluster failed")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if cr.ObjectMeta.DeletionTimestamp.IsZero() {
		// cr没有被删除, 如果没有被打上finalizer标签则添加
		if !controllerutil.ContainsFinalizer(cr, etcdClusterFinalizerName) {
			controllerutil.AddFinalizer(cr, etcdClusterFinalizerName)
			l.V(1).Info("add finalizer successfully")
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// cr被删除
		if controllerutil.ContainsFinalizer(cr, etcdClusterFinalizerName) {
			// TODO: 移除需要预先释放的资源

			controllerutil.RemoveFinalizer(cr, etcdClusterFinalizerName)
			l.V(1).Info("remove finalizer successfully")
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 获取cr下面关联的pod
	var err error
	switch cr.Status.Phase {
	case databasev1alpha1.ClusterPhaseNone:
		l.V(1).Info("cluster status is none, change to creating...")
		cr.Status.Phase = databasev1alpha1.ClusterPhaseCreating
		err = r.Status().Update(ctx, cr)
		if err != nil {
			l.Error(err, "update cluster status to creating failed")
		}
	case databasev1alpha1.ClusterPhaseCreating:
		l.V(1).Info("cluster status is creating, sync cluster status")
	case databasev1alpha1.ClusterPhaseRunning:
		l.V(1).Info("cluster status is running, stop reconcile")
	case databasev1alpha1.ClusterPhaseFailed:
		l.V(1).Info("cluster status is failed, stop reconcile")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.EtcdCluster{}).
		Complete(r)
}
