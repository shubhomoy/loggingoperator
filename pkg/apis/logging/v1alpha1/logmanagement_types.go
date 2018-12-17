package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogManagementSpec defines the desired state of LogManagement
type LogManagementSpec struct {
	FluentBitLogFile       string   `json:"fluentbit-logfile"`
	K8sMetadata            bool     `json:"include-k8s-metadata"`
	ESKibanaVersion        string   `json:"es-kib-version"`
	LogManagementNamespace string   `json:"namespace"`
	Inputs                 []Input  `json:"inputs"`
	Parsers                []Parser `json:"parsers"`
	Outputs                []Output `json:"outputs"`
	ElasticSearchRequired  bool     `json:"elasticsearch"`
	KibanaRequired         bool     `json:"kibana"`
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

// Input adds input spec
type Input struct {
	DeploymentName string `json:"deployment-name"`
	Tag            string `json:"tag"`
	IndexPattern   string `json:"index-pattern"`
}

// Parser adds
type Parser struct {
	Name     string `json:"name"`
	Regex    string `json:"regex,omitempty"`
	Selector string `json:"selector"`
}

// Output spec
type Output struct {
	Type string `json:"type"`
}

func init() {
	SchemeBuilder.Register(&LogManagement{}, &LogManagementList{})
}
