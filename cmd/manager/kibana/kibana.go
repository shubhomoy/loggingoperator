package kibana

import (
	"strconv"

	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createEnvironmentVariables(esService *corev1.Service) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "ELASTICSEARCH_URL",
			Value: "http://" + esService.Spec.ClusterIP + ":" + strconv.FormatInt(int64(esService.Spec.Ports[0].Port), 10),
		},
	}
}

// CreateKibanaDeployment - creates Kibana deployment
func CreateKibanaDeployment(cr *loggingv1alpha1.LogManagement, esService *corev1.Service) *extensionv1.Deployment {
	label := map[string]string{
		"app": "kibana",
	}
	return &extensionv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "kibana",
			Labels:    label,
			Namespace: "default",
		},

		Spec: extensionv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kibana",
							Image: "docker.elastic.co/kibana/kibana:" + cr.Spec.ESKibanaVersion,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5601,
								},
							},
							Env: createEnvironmentVariables(esService),
						},
					},
				},
			},
		},
	}
}

// CreateKibanaService - generates kibana service
func CreateKibanaService(cr *loggingv1alpha1.LogManagement) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "kibana",
			Namespace: "default",
		},

		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kibana",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 5601,
					TargetPort: intstr.IntOrString{
						IntVal: int32(5601),
					},
				},
			},
			Type: "LoadBalancer",
		},
	}
}
