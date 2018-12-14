package elasticsearch

import (
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	core1 "k8s.io/api/core/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateElasticsearchDeployment creates ES deployment
func CreateElasticsearchDeployment(cr *loggingv1alpha1.LogManagement) *extensionv1.Deployment {
	label := map[string]string{
		"run": "elasticsearch",
	}

	return &extensionv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "elasticsearch",
			Namespace: cr.Spec.LogManagementNamespace,
			Labels:    label,
		},

		Spec: extensionv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},

			Template: core1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},

				Spec: core1.PodSpec{
					Containers: []core1.Container{
						{
							Name:  "elasticsearch",
							Image: "docker.elastic.co/elasticsearch/elasticsearch:" + cr.Spec.ESKibanaVersion,
							Ports: []core1.ContainerPort{
								{
									ContainerPort: 9200,
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateElasticsearchService generates ES service
func CreateElasticsearchService(cr *loggingv1alpha1.LogManagement) *core1.Service {
	return &core1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "elasticsearch",
			Namespace: cr.Spec.LogManagementNamespace,
		},

		Spec: core1.ServiceSpec{
			Selector: map[string]string{
				"run": "elasticsearch",
			},
			Ports: []core1.ServicePort{
				{
					Port: 9200,
					TargetPort: intstr.IntOrString{
						IntVal: int32(9200),
					},
				},
			},
		},
	}
}
