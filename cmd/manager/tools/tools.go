package tools

import (
	"github.com/log_management/logging-operator/cmd/manager/tools/elasticsearch"
	"github.com/log_management/logging-operator/cmd/manager/tools/fluentbit"
	"github.com/log_management/logging-operator/cmd/manager/tools/fluentd"
	"github.com/log_management/logging-operator/cmd/manager/tools/kibana"
	"github.com/log_management/logging-operator/cmd/manager/utils"
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Defaults declares some default values
var Defaults = map[string]string{
	"FLUENTBIT-LOG-FILE":        "/var/log/fluentbit.log",
	"FLUENTBIT-SVC-ACOUNT-NAME": "fluent-bit",
	"FLUENTBIT-ROLE-NAME":       "fluent-bit-read",
}

// Tools structure declarations
type Tools struct {
	cr            *loggingv1alpha1.LogManagement
	FluentBit     FluentBit
	FluentD       FluentD
	ElasticSearch ElasticSearch
	Kibana        Kibana
}

func (t *Tools) init() {
	if t.cr.Spec.FluentBitLogFile == "" {
		t.cr.Spec.FluentBitLogFile = Defaults["FLUENTBIT-LOG-FILE"]
	}

	t.FluentBit = FluentBit{
		cr: t.cr,
	}

	t.FluentD = FluentD{
		cr: t.cr,
	}

	t.ElasticSearch = ElasticSearch{
		cr: t.cr,
	}

	t.Kibana = Kibana{
		cr: t.cr,
	}
}

// SetupAccountsAndBindings creates Service Account, Cluster Role and Cluster Binding for FluentBit DaemonSet
func (t *Tools) SetupAccountsAndBindings() (*corev1.Namespace, *corev1.ServiceAccount, *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding) {
	namespace := t.createNamespace()
	svcAccount := t.createServiceAccount()
	clusterRole := t.createClusterRole()
	roleBinding := t.createRoleBinding(clusterRole, svcAccount)

	t.FluentBit.serviceAccount = svcAccount
	return namespace, svcAccount, clusterRole, roleBinding
}

// FluentBit structure
type FluentBit struct {
	cr             *loggingv1alpha1.LogManagement
	serviceAccount *corev1.ServiceAccount
}

// GetConfigMap returns FluentBit ConfigMap
func (f FluentBit) GetConfigMap() (*corev1.ConfigMap, *corev1.ConfigMap) {
	return &corev1.ConfigMap{}, fluentbit.CreateConfigMap(f.cr)
}

// GetDaemonSet returns FluentBit DaemonSet
func (f FluentBit) GetDaemonSet() (*appsv1.DaemonSet, *appsv1.DaemonSet) {
	return &appsv1.DaemonSet{}, fluentbit.CreateDaemonSet(f.cr, f.serviceAccount)
}

// -------------------------------

// FluentD structure
type FluentD struct {
	cr *loggingv1alpha1.LogManagement
}

// GetConfigMap returns FluentD configmap
func (f FluentD) GetConfigMap() (*corev1.ConfigMap, *corev1.ConfigMap) {
	return &corev1.ConfigMap{}, fluentd.CreateConfigMap(f.cr)
}

// GetService returns FluentD service
func (f FluentD) GetService() (*corev1.Service, *corev1.Service) {
	return &corev1.Service{}, fluentd.CreateFluentDService(f.cr)
}

// GetDaemonSet returns FluentD DaemonSet
func (f FluentD) GetDaemonSet(esSpec *utils.ElasticSearchSpec) (*appsv1.Deployment, *appsv1.Deployment) {
	return &appsv1.Deployment{}, fluentd.CreateDaemonSet(f.cr, esSpec)
}

// -------------------------------

// ElasticSearch structure
type ElasticSearch struct {
	cr *loggingv1alpha1.LogManagement
}

// GetDeployment returns ES deployment
func (e ElasticSearch) GetDeployment() (*appsv1.Deployment, *appsv1.Deployment) {
	return &appsv1.Deployment{}, elasticsearch.CreateElasticsearchDeployment(e.cr)
}

// GetService returns ES service
func (e ElasticSearch) GetService() (*corev1.Service, *corev1.Service) {
	return &corev1.Service{}, elasticsearch.CreateElasticsearchService(e.cr)
}

// -------------------------------

// Kibana structure
type Kibana struct {
	cr *loggingv1alpha1.LogManagement
}

// GetDeployment returns Kibana Deployment
func (k *Kibana) GetDeployment(esSpec *utils.ElasticSearchSpec) (*appsv1.Deployment, *appsv1.Deployment) {
	return &appsv1.Deployment{}, kibana.CreateKibanaDeployment(k.cr, esSpec)
}

// GetService returns Kibana Service
func (k *Kibana) GetService() (*corev1.Service, *corev1.Service) {
	return &corev1.Service{}, kibana.CreateKibanaService(k.cr)
}

/* -------------------------------
// Util Fucntions
// ------------------------------- */
func (t Tools) createServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      Defaults["FLUENTBIT-SVC-ACOUNT-NAME"],
			Namespace: t.cr.ObjectMeta.Namespace,
		},
	}
}

func (t Tools) createClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: Defaults["FLUENTBIT-ROLE-NAME"],
		},

		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"namespaces", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
}

func (t Tools) createRoleBinding(clusterRole *rbacv1.ClusterRole, svcAccount *corev1.ServiceAccount) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: Defaults["FLUENTBIT-ROLE-NAME"],
		},

		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     clusterRole.TypeMeta.Kind,
			Name:     clusterRole.ObjectMeta.Name,
		},

		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      svcAccount.ObjectMeta.Name,
			Namespace: t.cr.ObjectMeta.Namespace,
		}},
	}
}

func (t Tools) createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: t.cr.ObjectMeta.Namespace,
		},
	}
}

// ------------------------------

// GetTools returns an instance of Tools
func GetTools(customResource *loggingv1alpha1.LogManagement) *Tools {
	tools := Tools{
		cr: customResource,
	}
	tools.init()
	return &tools
}
