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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLogManagement{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("logmanagement-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

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

	instance := &loggingv1alpha1.LogManagement{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Log Management instance not found!")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, nil
	}

	if len(instance.Spec.Watch) > 0 {
		updateSpec(instance)
	}

	esSpec := utils.ElasticSearchSpec{}
	tools := tools.GetTools(instance)

	_, svcAccount, role, binding := tools.SetupAccountsAndBindings()

	existingSvcAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svcAccount.Name, Namespace: svcAccount.Namespace}, existingSvcAccount)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Service Account")
		return createK8sObject(instance, svcAccount, r)
	}

	existingRole := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: role.Name}, existingRole)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Cluster Role")
		return createK8sObject(instance, role, r)
	}

	existingBinding := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: binding.Name}, existingBinding)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Role Binding")
		return createK8sObject(instance, binding, r)
	}

	existingfluentdService, fluentdService := tools.FluentD.GetService()
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentdService.Name, Namespace: fluentdService.Namespace}, existingfluentdService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Service")
		return createK8sObject(instance, fluentdService, r)
	}

	existingFluentBitConfigMap, configMap := tools.FluentBit.GetConfigMap()
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existingFluentBitConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit Config Map")
		return createK8sObject(instance, configMap, r)
	}

	existingFluentBitDaemonSet, fluentBitDaemonSet := tools.FluentBit.GetDaemonSet()
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentBitDaemonSet.Name, Namespace: fluentBitDaemonSet.Namespace}, existingFluentBitDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentBit DaemonSet")
		return createK8sObject(instance, fluentBitDaemonSet, r)
	}

	if instance.Spec.ElasticSearch.Required {
		existingES, elasticsearch := tools.ElasticSearch.GetDeployment()
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticsearch.Name, Namespace: elasticsearch.Namespace}, existingES)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating Elasticsearch")
			return createK8sObject(instance, elasticsearch, r)
		}

		existingElasticSearchService, elasticSearchService := tools.ElasticSearch.GetService()
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: elasticSearchService.Name, Namespace: elasticSearchService.Namespace}, existingElasticSearchService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating ES Service")
			return createK8sObject(instance, elasticSearchService, r)
		} else {
			esSpec.Host = existingElasticSearchService.Spec.ClusterIP
			esSpec.Port = strconv.FormatInt(int64(existingElasticSearchService.Spec.Ports[0].Port), 10)
		}
	} else {
		esSpec.Host = instance.Spec.ElasticSearch.Host
		esSpec.Port = instance.Spec.ElasticSearch.Port
	}

	existingKibana, kibana := tools.Kibana.GetDeployment(&esSpec)
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kibana.Name, Namespace: kibana.Namespace}, existingKibana)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Deployment")
		return createK8sObject(instance, kibana, r)
	}

	existingKibanaService, kibanaService := tools.Kibana.GetService()
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kibanaService.Name, Namespace: kibanaService.Namespace}, existingKibanaService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating Kibana Service")
		return createK8sObject(instance, kibanaService, r)
	}

	existingFluentDConfigMap, fluentDConfigMap := tools.FluentD.GetConfigMap()
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDConfigMap.Name, Namespace: fluentDConfigMap.Namespace}, existingFluentDConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD Config Map")
		return createK8sObject(instance, fluentDConfigMap, r)
	}

	existingFluentD, fluentDDaemonSet := tools.FluentD.GetDaemonSet(&esSpec)
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fluentDDaemonSet.Name, Namespace: fluentDDaemonSet.Namespace}, existingFluentD)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating FluentD DaemonSet")
		return createK8sObject(instance, fluentDDaemonSet, r)
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
		reqLogger.Info("FluentD Config updated")
		if err != nil {
			reqLogger.Error(err, "Failed")
		} else {
			err = r.client.Delete(context.TODO(), existingFluentD)
			return reconcile.Result{Requeue: true}, nil
		}
	}

	return reconcile.Result{}, nil
}

func createK8sObject(instance *loggingv1alpha1.LogManagement, obj v1.Object, r *ReconcileLogManagement) (reconcile.Result, error) {
	var err error
	err = controllerutil.SetControllerReference(instance, obj, r.scheme)

	switch t := obj.(type) {
	case *corev1.ServiceAccount:
		err = r.client.Create(context.TODO(), t)
	case *rbacv1.ClusterRole:
		err = r.client.Create(context.TODO(), t)
	case *rbacv1.ClusterRoleBinding:
		err = r.client.Create(context.TODO(), t)
	case *corev1.Namespace:
		err = r.client.Create(context.TODO(), t)
	case *corev1.ConfigMap:
		err = r.client.Create(context.TODO(), t)
	case *corev1.Service:
		err = r.client.Create(context.TODO(), t)
	case *extensionv1.DaemonSet:
		err = r.client.Create(context.TODO(), t)
	case *extensionv1.Deployment:
		err = r.client.Create(context.TODO(), t)
	}
	if err != nil {
		fmt.Println(err)
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
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
