package fluentd

import (
	"bytes"
	"strconv"
	"text/template"

	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateFluentDService - create fluentd service
func CreateFluentDService(cr *loggingv1alpha1.LogManagement) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd",
			Namespace: cr.Spec.LogManagementNamespace,
		},

		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"run": "fluentd",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8777,
					TargetPort: intstr.IntOrString{
						IntVal: int32(8777),
					},
				},
			},
		},
	}
}

// CreateConfigMap for FluentD
func CreateConfigMap(cr *loggingv1alpha1.LogManagement) *corev1.ConfigMap {

	templateInput := TemplateInput{}

	for _, i := range cr.Spec.Inputs {
		var outputs []Output
		for _, o := range i.Outputs {
			outputs = append(outputs, Output{Type: o.Type, IndexPattern: o.IndexPattern})
		}
		templateInput.Inputs = append(templateInput.Inputs, Input{DeploymentName: i.DeploymentName, Tag: i.Tag, Outputs: outputs})
	}

	configMap := generateConfig(templateInput, configmapTemplate)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-config",
			Namespace: cr.Spec.LogManagementNamespace,
		},

		Data: map[string]string{
			"fluent.conf": *configMap,
		},
	}
}

// CreateDaemonSet creates daemonset for FluentBit
func CreateDaemonSet(cr *loggingv1alpha1.LogManagement, esService *corev1.Service) *extensionv1.Deployment {
	labels := map[string]string{
		"run": "fluentd",
	}
	return &extensionv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd",
			Namespace: cr.Spec.LogManagementNamespace,
			Labels:    labels},
		Spec: extensionv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "fluentd",
							Image:           "jc-fluentd:v1",
							ImagePullPolicy: "IfNotPresent",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 8777,
								},
							},
							Env:          generateEnvironmentVariables(esService),
							VolumeMounts: generateVolumeMounts(),
						},
					},
					Volumes: generateVolumes(),
				},
			},
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
			Name:      "config-volume",
			MountPath: "/fluentd/etc/fluent.conf",
			SubPath:   "fluent.conf",
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
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "fluentd-config",
					},
				},
			},
		},
	}
}

func generateEnvironmentVariables(esService *corev1.Service) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "FLUENTD_CONF",
			Value: "fluent.conf",
		},
		{
			Name:  "ES_PORT",
			Value: strconv.FormatInt(int64(esService.Spec.Ports[0].Port), 10),
		},
		{
			Name:  "ES_HOST",
			Value: esService.Spec.ClusterIP,
		},
	}
}

// TemplateInput structure
type TemplateInput struct {
	Inputs []Input
}

// Input structure
type Input struct {
	DeploymentName string
	Tag            string
	Outputs        []Output
}

// Output spec
type Output struct {
	Type         string
	IndexPattern string
}

func generateConfig(TemplateInput TemplateInput, configmapTemplate string) *string {
	output := new(bytes.Buffer)
	tmpl, err := template.New("config").Parse(configmapTemplate)
	if err != nil {
		return nil
	}
	err = tmpl.Execute(output, TemplateInput)
	outputString := output.String()
	return &outputString
}
