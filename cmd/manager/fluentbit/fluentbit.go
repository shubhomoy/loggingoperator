package fluentbit

import (
	"bytes"
	"errors"
	"text/template"

	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validateConfigMap(spec loggingv1alpha1.LogManagementSpec) error {
	for _, in := range spec.Inputs {
		present := false
		for _, par := range spec.Parsers {
			if in.Parser == par.Name {
				present = true
				break
			}
		}
		if !present {
			return errors.New("Parser not found")
		}
	}
	return nil
}

func generateEnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "FLUENT_ELASTICSEARCH_HOST",
			Value: "10.98.50.241",
		},
		{
			Name:  "FLUENT_ELASTICSEARCH_PORT",
			Value: "30240",
		},
	}
}

func generateVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "varlog",
			MountPath: "/var/log",
		},
		{
			Name:      "fluent-bit-config",
			MountPath: "/fluent-bit/etc/",
		},
		{
			Name:      "varlibcontainers",
			ReadOnly:  true,
			MountPath: "/var/lib/docker/containers",
		},
	}
}

func generateVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "fluent-bit-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "fluent-bit-config",
					},
				},
			},
		},
		{
			Name: "varlibcontainers",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker/containers",
				},
			},
		},
	}
}

// CreateDaemonSet generates the FluentBit DS
func CreateDaemonSet(cr *loggingv1alpha1.LogManagement, serviceAccount corev1.ServiceAccount) *extensionv1.DaemonSet {
	labels := map[string]string{
		"k8s-app":                       "fluent-bit-logging",
		"kubernetes.io/cluster-service": "true",
	}
	return &extensionv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: "default",
			Labels:    labels},
		Spec: extensionv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "fluent-bit",
							Image:           "fluent/fluent-bit:0.14.6",
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 2020,
								},
							},
							Env:          generateEnvironmentVariables(),
							VolumeMounts: generateVolumeMounts(),
						},
					},
					Volumes:            generateVolumes(),
					ServiceAccountName: serviceAccount.Name,
					Tolerations: []corev1.Toleration{
						corev1.Toleration{
							Key:      "node-role.kubernetes.io/master",
							Operator: "Exists",
							Effect:   "NoSchedule",
						},
					},
				},
			},
		},
	}
}

// CreateServiceAccount - Generate service account for FluentBit
func CreateServiceAccount(cr *loggingv1alpha1.LogManagement) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: "default",
		},
	}
}

// CreateClusterRole - create cluster role for fluentbit
func CreateClusterRole(cr *loggingv1alpha1.LogManagement) *rbacv1.ClusterRole {
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

// CreateRoleBinding - create role binding
func CreateRoleBinding(cr *loggingv1alpha1.LogManagement) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit-read",
			Namespace: "default",
		},

		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fluent-bit-read",
		},

		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "fluent-bit",
			Namespace: "default",
		}},
	}
}

// CreateConfigMap - generate config map
func CreateConfigMap(cr *loggingv1alpha1.LogManagement) (*corev1.ConfigMap, error) {

	err := validateConfigMap(cr.Spec)

	if err != nil {
		return nil, err
	}

	templateInput := TemplateInput{
		FluentBitLogFile: cr.Spec.FluentBitLogFile,
		K8sMetadata:      cr.Spec.K8sMetadata,
	}
	for _, i := range cr.Spec.Inputs {
		templateInput.Inputs = append(templateInput.Inputs, Input{DeploymentName: i.DeploymentName, Tag: i.Tag, Parser: i.Parser})
	}

	for _, i := range cr.Spec.Parsers {
		templateInput.Parsers = append(templateInput.Parsers, Parser{Name: i.Name, Regex: i.Regex})
	}

	configMap, err := generateConfig(templateInput, configmapTemplate)
	parserMap, err := generateConfig(templateInput, parsersTemplate)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-config",
			Labels: map[string]string{
				"k8s-app": "fluent-bit",
			},
			Namespace: "default",
		},

		Data: map[string]string{
			"fluent-bit.conf": *configMap,
			"parsers.conf":    *parserMap,
		},
	}, nil
}

// TemplateInput defines the input template placeholder
type TemplateInput struct {
	FluentBitLogFile string
	K8sMetadata      bool
	Inputs           []Input
	Parsers          []Parser
}

// Input defines the structure of input placeholder
type Input struct {
	DeploymentName string
	Tag            string
	Parser         string
}

// Parser defines structure of Parsers
type Parser struct {
	Name  string
	Regex string
}

func generateConfig(input TemplateInput, templateFile string) (*string, error) {
	output := new(bytes.Buffer)
	tmpl, err := template.New("config").Parse(templateFile)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(output, input)
	if err != nil {
		return nil, err
	}
	outputString := output.String()
	return &outputString, nil
}
