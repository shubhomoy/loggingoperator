package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogManagementSpec defines the desired state of LogManagement
type LogManagementSpec struct {
	FluentBitLogFile string        `json:"fluentbit-logfile"`
	K8sMetadata      bool          `json:"include-k8s-metadata"`
	ESKibanaVersion  string        `json:"es-kib-version"`
	Watch            []Watch       `json:"watch"`
	Parsers          []Parser      `json:"parsers"`
	ElasticSearch    ElasticSearch `json:"elasticsearch-spec"`
	KibanaRequired   bool          `json:"kibana"`
}

// LogManagementStatus defines the observed state of LogManagement
type LogManagementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LogManagement is the Schema for the logmanagements API
// +k8s:openapi-gen=true
type LogManagement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogManagementSpec   `json:"spec,omitempty"`
	Status LogManagementStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LogManagementList contains a list of LogManagement
type LogManagementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogManagement `json:"items"`
}

// Parser adds
type Parser struct {
	Name  string `json:"name"`
	Regex string `json:"regex,omitempty"`
}

// InputParser defines input parser structure
type InputParser struct {
	Name string `json:"name"`
}

// Output spec
type Output struct {
	Type         string `json:"type"`
	IndexPattern string `json:"index-pattern"`
}

// Watch spec
type Watch struct {
	Namespace   string        `json:"namespace"`
	Deployments []Deployment  `json:"deployments"`
	Parsers     []InputParser `json:"parsers"`
	Outputs     []Output      `json:"outputs"`
	Tag         string
}

// ElasticSearch spec
type ElasticSearch struct {
	Required   bool   `json:"required"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	HTTPS      bool   `json:"https"`
	HTTPString string
}

// Deployment spec
type Deployment struct {
	Name string `json:"name"`
}

func init() {
	SchemeBuilder.Register(&LogManagement{}, &LogManagementList{})
}
