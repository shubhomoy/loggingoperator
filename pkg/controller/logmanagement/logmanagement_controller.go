package logmanagement

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/log_management/logging-operator/cmd/manager/tools"
	"github.com/log_management/logging-operator/cmd/manager/utils"
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
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &loggingv1alpha1.LogManagement{},
	})
	err = c.Watch(&source.Kind{Type: &extensionv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
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
	// fmt.Println("-------------------------------------------------------------------")
	// fmt.Println(request.NamespacedName)
	// fmt.Println("-------------------------------------------------------------------")

	// Fetch the LogManagement instance
	instance := &loggingv1alpha1.LogManagement{}

	// reqLogger.Info("Checking Log Management Instance")
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	// reqLogger.Info("Checking Log Management Instance - DONE")
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Log Management instance not found!")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, nil
	}

	if request.Namespace != instance.Spec.LogManagementNamespace {
		reqLogger.Info("Namespace mismatch")
		return reconcile.Result{}, nil
	}

	if len(instance.Spec.Watch) > 0 {
		updateSpec(instance)
	}

	// ------------------------------------
	/* Accounts, Roles and Binding Setup
	// ------------------------------------
	*/
	esSpec := utils.ElasticSearchSpec{}
	tools := tools.GetTools(instance)

	svcAccount, role, binding := tools.SetupAccountsAndBindings()

	// reqLogger.Info("Checking Namespace")
	// existingNamespace := &corev1.Namespace{}
	// err = r.client.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, existingNamespace)
	// reqLogger.Info("Checking Namespace - DONE")
	// if err != nil && errors.IsNotFound(err) {
	// 	reqLogger.Info("Creating Namespace")
	// 	CreateK8sObject(instance, namespace, r)
	// 	return reconcile.Result{Requeue: true}, nil
	// }

	// reqLogger.Info("Checking Service Acouunt")
	existingSvcAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svcAccount.Name, Namespace: svcAccount.Namespace}, existingSvcAccount)
	// reqLogger.Info("Checking Service Acouunt - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Service Account")
		CreateK8sObject(instance, svcAccount, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// reqLogger.Info("Checking CLuster Role")
	existingRole := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: role.Name}, existingRole)
	// reqLogger.Info("Checking Cluster Role - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Cluster Role")
		CreateK8sObject(instance, role, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// reqLogger.Info("Checking Role Binding")
	existingBinding := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: binding.Name}, existingBinding)
	// reqLogger.Info("Checking Role Binding - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Role Binding")
		CreateK8sObject(instance, binding, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// ------------------------------------

	// Creating FluentD service
	// reqLogger.Info("Checking FluentD Service")
	fluentdService := tools.FluentD.GetService()
	existingfluentdService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentdService.Name, Namespace: fluentdService.Namespace}, existingfluentdService)
	// reqLogger.Info("Checking FluentD Service - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Service")
		CreateK8sObject(instance, fluentdService, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// Creating FluentBit Config Map
	// reqLogger.Info("Checking FluentBit ConfigMap")
	configMap := tools.FluentBit.GetConfigMap()
	existingFluentBitConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existingFluentBitConfigMap)
	// reqLogger.Info("Checking FluentBit ConfigMap - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit Config Map")
		CreateK8sObject(instance, configMap, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// Creating FluentBit DS
	// reqLogger.Info("Checking FluentBit DS")
	fluentBitDaemonSet := tools.FluentBit.GetDaemonSet()
	existingFluentBitDaemonSet := &extensionv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentBitDaemonSet.Name, Namespace: fluentBitDaemonSet.Namespace}, existingFluentBitDaemonSet)
	// reqLogger.Info("Checking FluentBit DS - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit DaemonSet")
		CreateK8sObject(instance, fluentBitDaemonSet, r)
		return reconcile.Result{Requeue: true}, nil
	}

	if instance.Spec.ElasticSearch.Required {
		// Creating ES
		// reqLogger.Info("Checking ElasticSearch")
		elasticsearch := tools.ElasticSearch.GetDeployment()
		existingES := &extensionv1.Deployment{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticsearch.Name, Namespace: elasticsearch.Namespace}, existingES)
		// reqLogger.Info("Checking ElasticSearch - DONE")
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating Elasticsearch")
			CreateK8sObject(instance, elasticsearch, r)
			return reconcile.Result{Requeue: true}, nil
		}

		// Creating ES service
		// reqLogger.Info("Checking ElasticSearch Service")
		elasticSearchService := tools.ElasticSearch.GetService()
		existingElasticSearchService := &corev1.Service{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticSearchService.Name, Namespace: elasticSearchService.Namespace}, existingElasticSearchService)
		// reqLogger.Info("Checking ElasticSearch Service - DONE")
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating ES Service")
			CreateK8sObject(instance, elasticSearchService, r)
			return reconcile.Result{Requeue: true}, nil
		}

		esSpec.Host = elasticSearchService.Spec.ClusterIP
		esSpec.Port = strconv.FormatInt(int64(elasticSearchService.Spec.Ports[0].Port), 10)
	} else {
		esSpec.Host = instance.Spec.ElasticSearch.Host
		esSpec.Port = instance.Spec.ElasticSearch.Port
	}

	// Creating Kibana deployment
	// reqLogger.Info("Checking Kibana")
	kibana := tools.Kibana.GetDeployment(&esSpec)
	existingKibana := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kibana.Name, Namespace: kibana.Namespace}, existingKibana)
	// reqLogger.Info("Checking Kibana - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Deployment")
		CreateK8sObject(instance, kibana, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// Creating Kibana service
	// reqLogger.Info("Checking Kibana Service")
	kibanaService := tools.Kibana.GetService()
	existingKibanaService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kibanaService.Name, Namespace: kibanaService.Namespace}, existingKibanaService)
	// reqLogger.Info("Checking Kibana Service - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Service")
		CreateK8sObject(instance, kibanaService, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// Creating FluentD ConfigMap
	// reqLogger.Info("Checking FluentD ConfigMap")
	fluentDConfigMap := tools.FluentD.GetConfigMap()
	existingFluentDConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDConfigMap.Name, Namespace: fluentDConfigMap.Namespace}, existingFluentDConfigMap)
	// reqLogger.Info("Checking FluentD ConfigMap - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Config Map")
		CreateK8sObject(instance, fluentDConfigMap, r)
		return reconcile.Result{Requeue: true}, nil
	}

	// Creating FluentD DS
	// reqLogger.Info("Checking FluentD DS")
	fluentDDaemonSet := tools.FluentD.GetDaemonSet(&esSpec)
	existingFluentD := &extensionv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDDaemonSet.Name, Namespace: fluentDDaemonSet.Namespace}, existingFluentD)
	// reqLogger.Info("Checking FluentD DS - DONE")
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD DaemonSet")
		CreateK8sObject(instance, fluentDDaemonSet, r)
		return reconcile.Result{Requeue: true}, nil
	}

	/*
		// Updation
	*/

	// FluentBit Config Update
	eq := reflect.DeepEqual(existingFluentBitConfigMap.Data, configMap.Data)
	if !eq {
		reqLogger.Info("FluentBit Config Changed. Updating...")
		existingFluentBitConfigMap.Data = configMap.Data
		err = r.client.Update(context.TODO(), existingFluentBitConfigMap)
		reqLogger.Info("FluentBit Config Updated.")
		if err != nil {
			reqLogger.Error(err, "Failed")
		} else {
			err = r.client.Delete(context.TODO(), existingFluentBitDaemonSet)
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// FluentD config update
	eq = reflect.DeepEqual(existingFluentDConfigMap.Data, fluentDConfigMap.Data)
	if !eq {
		reqLogger.Info("FluentD Config Changed. Updating...")
		existingFluentDConfigMap.Data = fluentDConfigMap.Data
		err = r.client.Update(context.TODO(), existingFluentDConfigMap)
		reqLogger.Info("FluentD Config Changed. Updating...")
		if err != nil {
			reqLogger.Error(err, "Failed")
		} else {
			err = r.client.Delete(context.TODO(), existingFluentD)
			return reconcile.Result{Requeue: true}, nil
		}
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

func updateSpec(cr *loggingv1alpha1.LogManagement) {
	var inputs []loggingv1alpha1.Input
	for _, watcher := range cr.Spec.Watch {
		input := loggingv1alpha1.Input{
			DeploymentName: "*_" + watcher.Namespace + "_*",
			Tag:            watcher.Namespace,
			Parsers:        watcher.Parsers,
			Outputs:        watcher.Outputs,
		}

		inputs = append(inputs, input)
	}
	cr.Spec.Watch = nil
	cr.Spec.Inputs = inputs
}
