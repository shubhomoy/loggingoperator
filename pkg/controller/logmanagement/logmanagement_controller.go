package logmanagement

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/log_management/logging-operator/cmd/manager/tools"
	"github.com/log_management/logging-operator/cmd/manager/tools/fluentd"
	"github.com/log_management/logging-operator/cmd/manager/tools/kibana"
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
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

	err = c.Watch(&source.Kind{Type: &extensionv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &loggingv1alpha1.LogManagement{},
	})

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

	// ------------------------------------
	/* Accounts, Roles and Binding Setup
	// ------------------------------------
	*/

	tools := tools.GetTools(instance)

	namespace, svcAccount, role, binding := tools.SetupAccountsAndBindings()

	existingNamespace := &corev1.Namespace{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, existingNamespace)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Namespace")
		CreateK8sObject(instance, namespace, r)
	}

	existingSvcAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svcAccount.Name, Namespace: svcAccount.Namespace}, existingSvcAccount)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Service Account")
		CreateK8sObject(instance, svcAccount, r)
	}

	existingRole := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: role.Name}, existingRole)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Cluster Role")
		CreateK8sObject(instance, role, r)
	}

	existingBinding := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: binding.Name}, existingBinding)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Role Binding")
		CreateK8sObject(instance, binding, r)
	}

	// ------------------------------------

	// Creating FluentD service
	fluentdService := tools.FluentD.GetService()
	existingfluentdService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentdService.Name, Namespace: fluentdService.Namespace}, existingfluentdService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Service")
		CreateK8sObject(instance, fluentdService, r)
	}

	// Creating Config Map
	configMap := tools.FluentBit.GetConfigMap()
	existingFluentBitConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existingFluentBitConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit Config Map")
		CreateK8sObject(instance, configMap, r)
	}

	fluentBitDaemonSet := tools.FluentBit.GetDaemonSet()
	existingFluentBitDaemonSet := &extensionv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentBitDaemonSet.Name, Namespace: fluentBitDaemonSet.Namespace}, existingFluentBitDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit DaemonSet")
		CreateK8sObject(instance, fluentBitDaemonSet, r)
	}

	// Creating ES
	elasticsearch := tools.ElasticSearch.GetDeployment()
	existingES := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticsearch.Name, Namespace: elasticsearch.Namespace}, existingES)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Elasticsearch")
		CreateK8sObject(instance, elasticsearch, r)
	}

	// Creating ES service
	elasticSearchService := tools.ElasticSearch.GetService()
	existingElasticSearchService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticSearchService.Name, Namespace: elasticSearchService.Namespace}, existingElasticSearchService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating ES Service")
		CreateK8sObject(instance, elasticSearchService, r)
	}

	// Creating Kibana deployment
	kib := kibana.CreateKibanaDeployment(instance, elasticSearchService)
	kibanaFound := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kib.Name, Namespace: kib.Namespace}, kibanaFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Deployment")
		err = controllerutil.SetControllerReference(instance, kib, r.scheme)
		err = r.client.Create(context.TODO(), kib)
	}

	// Creating Kibana service
	kibanaService := kibana.CreateKibanaService(instance)
	kibanaServiceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kibanaService.Name, Namespace: kibanaService.Namespace}, kibanaServiceFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Service")
		err = controllerutil.SetControllerReference(instance, kibanaService, r.scheme)
		err = r.client.Create(context.TODO(), kibanaService)
	}

	fluentDConfigMap := tools.FluentD.GetConfigMap()
	existingFluentDConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDConfigMap.Name, Namespace: fluentDConfigMap.Namespace}, existingFluentDConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Config Map")
		CreateK8sObject(instance, fluentDConfigMap, r)
	}

	// Creating FluentD DS
	fluentDDaemonSet := fluentd.CreateDaemonSet(instance, elasticSearchService)
	foundFluentDDs := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDDaemonSet.Name, Namespace: fluentDDaemonSet.Namespace}, foundFluentDDs)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD DaemonSet")
		err = controllerutil.SetControllerReference(instance, fluentDDaemonSet, r.scheme)
		err = r.client.Create(context.TODO(), fluentDDaemonSet)
	}

	return reconcile.Result{Requeue: true}, nil
}

// CreateK8sObject creates K8s object
func CreateK8sObject(instance *loggingv1alpha1.LogManagement, obj interface{}, r *ReconcileLogManagement) {
	var err error
	switch obj.(type) {
	case *corev1.ServiceAccount:
		k8sObj := obj.(*corev1.ServiceAccount)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *rbacv1.ClusterRole:
		k8sObj := obj.(*rbacv1.ClusterRole)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *rbacv1.ClusterRoleBinding:
		k8sObj := obj.(*rbacv1.ClusterRoleBinding)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *corev1.Namespace:
		k8sObj := obj.(*corev1.Namespace)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *corev1.ConfigMap:
		k8sObj := obj.(*corev1.ConfigMap)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *corev1.Service:
		k8sObj := obj.(*corev1.Service)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *extensionv1.DaemonSet:
		k8sObj := obj.(*extensionv1.DaemonSet)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	case *extensionv1.Deployment:
		k8sObj := obj.(*extensionv1.Deployment)
		err = controllerutil.SetControllerReference(instance, k8sObj, r.scheme)
		err = r.client.Create(context.TODO(), k8sObj)
	}
	if err != nil {
		fmt.Println(err)
	}
}
