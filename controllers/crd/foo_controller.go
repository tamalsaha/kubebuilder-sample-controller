/*
Copyright 2021.

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

package crd

import (
	"context"
	crdv1alpha1 "github.com/tamalsaha/kubebuilder-sample-controller/apis/crd/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crd.learnkube.com,resources=foos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.learnkube.com,resources=foos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.learnkube.com,resources=foos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Foo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// your logic here
	var foo crdv1alpha1.Foo
	if err := r.Get(ctx, req.NamespacedName, &foo); err != nil {
		log.Error(err, "unable to fetch Foo")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// create or update deployment
	var deploy appsv1.Deployment

	depName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      foo.Spec.DeploymentName,
	}
	if err := r.Get(ctx, depName, &deploy); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		deploy = appsv1.Deployment{}
		// set owner
		if err := r.Create(ctx, &deploy); err != nil {
			log.Error(err, "unable to create deployment", "key", depName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	} else {
		// check matches owner
		if deploy.Spec.Replicas != foo.Spec.Replicas {
			deploy.Spec.Replicas = foo.Spec.Replicas
			if err := r.Update(ctx, &deploy); err != nil {
				log.Error(err, "unable to update deployment", "key", depName)
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}

		foo.Status.AvailableReplicas = deploy.Status.AvailableReplicas
		_ = r.Status().Update(ctx, &foo)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {
	configMapHandler := handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
		foos := &crdv1alpha1.FooList{}
		if err := r.List(context.Background(), foos, client.InNamespace(a.GetNamespace())); err != nil {
			return nil
		}
		var req []reconcile.Request
		for _, c := range foos.Items {
			if c.Spec.SecretName == a.GetName() {
				req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&c)})
			}
		}
		return req
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1alpha1.Foo{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, configMapHandler).
		Complete(r)
}
