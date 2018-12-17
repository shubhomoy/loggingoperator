package tools

import (
	"github.com/log_management/logging-operator/cmd/manager/tools/elasticsearch"
	"github.com/log_management/logging-operator/cmd/manager/tools/fluentbit"
	"github.com/log_management/logging-operator/cmd/manager/tools/fluentd"
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Defaults declares some default values
var Defaults = map[string]string{
	"FLUENTBIT-LOG-FILE": "/var/log/fluentbit.log",
	"NAMESPACE":          "logging",
}

// Tools structure declarations
type Tools struct {
	cr            *loggingv1alpha1.LogManagement
	FluentBit     FluentBit
	FluentD       FluentD
	ElasticSearch ElasticSearch
}

func (t *Tools) init() {
	if t.cr.Spec.FluentBitLogFile == "" {
		t.cr.Spec.FluentBitLogFile = Defaults["FLUENTBIT-LOG-FILE"]
	}

	if t.cr.Spec.LogManagementNamespace == "" {
		t.cr.Spec.LogManagementNamespace = Defaults["NAMESPACE"]
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
}

// SetupAccountsAndBindings creates Service Account, Cluster Role and Cluster Binding for FluentBit DaemonSet
func (t *Tools) SetupAccountsAndBindings() (*corev1.Namespace, *corev1.ServiceAccount, *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding) {
	namespace := t.createNamespace()
	svcAccount := t.createServiceAccount()
	clusterRole := t.createClusterRole()
	roleBinding := t.createRoleBinding()

	t.FluentBit.serviceAccount = svcAccount
	return namespace, svcAccount, clusterRole, roleBinding
}

// FluentBit structure
type FluentBit struct {
	cr             *loggingv1alpha1.LogManagement
	serviceAccount *corev1.ServiceAccount
}

// GetConfigMap returns FluentBit ConfigMap
func (f FluentBit) GetConfigMap() *corev1.ConfigMap {
	return fluentbit.CreateConfigMap(f.cr)
}

// GetDaemonSet returns FluentBit DaemonSet
func (f FluentBit) GetDaemonSet() *extensionv1.DaemonSet {
	return fluentbit.CreateDaemonSet(f.cr, f.serviceAccount)
}

// -------------------------------

// FluentD structure
type FluentD struct {
	cr *loggingv1alpha1.LogManagement
}

// GetConfigMap returns FluentD configmap
func (f FluentD) GetConfigMap() *corev1.ConfigMap {
	return fluentd.CreateConfigMap(f.cr)
}

// GetService returns FluentD service
func (f FluentD) GetService() *corev1.Service {
	return fluentd.CreateFluentDService(f.cr)
}

// GetDaemonSet returns FluentD DaemonSet
// func (f FluentD) GetDaemonSet() *extensionv1.DaemonSet {
// 	return fluentd.CreateDaemonSet(f.cr, f.serviceAccount)
// }

// -------------------------------

// ElasticSearch structure
type ElasticSearch struct {
	cr *loggingv1alpha1.LogManagement
}

// GetDeployment returns ES deployment
func (e ElasticSearch) GetDeployment() *extensionv1.Deployment {
	return elasticsearch.CreateElasticsearchDeployment(e.cr)
}

// GetService returns ES service
func (e ElasticSearch) GetService() *corev1.Service {
	return elasticsearch.CreateElasticsearchService(e.cr)
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
			Name:      "fluent-bit",
			Namespace: t.cr.Spec.LogManagementNamespace,
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
			Name: "fluent-bit-read",
		},

		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"namespaces", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
}

func (t Tools) createRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-read",
		},

		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fluent-bit-read",
		},

		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "fluent-bit",
			Namespace: t.cr.Spec.LogManagementNamespace,
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
			Name: t.cr.Spec.LogManagementNamespace,
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
