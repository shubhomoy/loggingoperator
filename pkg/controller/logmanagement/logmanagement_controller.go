package logmanagement

import (
	"context"

	"github.com/log_management/logging-operator/cmd/manager/elasticsearch"
	"github.com/log_management/logging-operator/cmd/manager/fluentbit"
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_logmanagement")

// Add creates a new LogManagement Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLogManagement{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("logmanagement-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LogManagement
	err = c.Watch(&source.Kind{Type: &loggingv1alpha1.LogManagement{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner LogManagement
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &loggingv1alpha1.LogManagement{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileLogManagement{}

// ReconcileLogManagement reconciles a LogManagement object
type ReconcileLogManagement struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a LogManagement object and makes changes based on the state read
// and what is in the LogManagement.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLogManagement) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling LogManagement")

	// Fetch the LogManagement instance
	instance := &loggingv1alpha1.LogManagement{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Creating Service Account
	serviceAccount := fluentbit.CreateServiceAccount(instance)
	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Service Account")
		err = controllerutil.SetControllerReference(instance, serviceAccount, r.scheme)
		err = r.client.Create(context.TODO(), serviceAccount)
	}

	// Creating Cluster Role
	clusterRole := fluentbit.CreateClusterRole(instance)
	foundClusterRole := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: clusterRole.Name}, foundClusterRole)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Cluster Role")
		err = controllerutil.SetControllerReference(instance, clusterRole, r.scheme)
		err = r.client.Create(context.TODO(), clusterRole)
	}

	// Creating Role Binding
	roleBinding := fluentbit.CreateRoleBinding(instance)
	foundRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: roleBinding.Name}, foundRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Role Binding")
		err = controllerutil.SetControllerReference(instance, roleBinding, r.scheme)
		err = r.client.Create(context.TODO(), roleBinding)
	}

	// Creating Config Map
	cm, err := fluentbit.CreateConfigMap(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	foundConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Config Map")
		err = controllerutil.SetControllerReference(instance, cm, r.scheme)
		err = r.client.Create(context.TODO(), cm)
	}

	// Creating DaemonSet
	daemonset := fluentbit.CreateDaemonSet(instance, *serviceAccount)
	foundDaemonSet := &extensionv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: daemonset.Name, Namespace: daemonset.Namespace}, foundDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating DaemonSet")
		err = controllerutil.SetControllerReference(instance, daemonset, r.scheme)
		err = r.client.Create(context.TODO(), daemonset)
	}

	// Creating ES
	es := elasticsearch.CreateElasticsearchDeployment(instance)
	esFound := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: es.Name, Namespace: es.Namespace}, esFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Elasticsearch")
		err = controllerutil.SetControllerReference(instance, es, r.scheme)
		err = r.client.Create(context.TODO(), es)
	}

	// Creating ES service
	esService := elasticsearch.CreateElasticsearchService(instance)
	esServiceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: esService.Name, Namespace: esService.Namespace}, esServiceFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Elasticsearch Service")
		err = controllerutil.SetControllerReference(instance, esService, r.scheme)
		err = r.client.Create(context.TODO(), esService)
	}

	return reconcile.Result{Requeue: true}, nil
}
