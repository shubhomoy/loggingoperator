package utils

import (
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
)

// ElasticSearchSpec spec
type ElasticSearchSpec struct {
	CurrentHost string
	CurrentPort string
	PrevHost    string
	PrevPort    string
	HTTPString  string
}

// Validator defines validator struct
type Validator struct {
	Validated    bool
	ErrorMessage string
}

// Validate validates LogManagement Custom Resource
func (v *Validator) Validate(cr *loggingv1alpha1.LogManagement) {
	v.Validated = true

	if cr.ObjectMeta.Namespace == "default" {
		v.ErrorMessage = v.ErrorMessage + "Please use LogManagement operator namespace other than default\n"
		v.Validated = false
	}

	if !cr.Spec.ElasticSearch.Required {
		if cr.Spec.ElasticSearch.Host == "" {
			v.ErrorMessage = v.ErrorMessage + "ElasticSearch host missing\n"
			v.Validated = false
		}
		if cr.Spec.ElasticSearch.Port == "" {
			v.ErrorMessage = v.ErrorMessage + "ElasticSearch port missing\n"
			v.Validated = false
		}
	} else {
		cr.Spec.ElasticSearch.HTTPS = false
	}

	if cr.Spec.ElasticSearch.HTTPS {
		cr.Spec.ElasticSearch.HTTPString = "https://"
	} else {
		cr.Spec.ElasticSearch.HTTPString = "http://"
	}

	if cr.Spec.ESKibanaVersion == "" {
		v.ErrorMessage = v.ErrorMessage + "es-kib-version missing\n"
		v.Validated = false
	}

	if len(cr.Spec.Watch) == 0 {
		v.ErrorMessage = v.ErrorMessage + "Watch section missing\n"
		v.Validated = false
	}

	parserNameList := make(map[string]bool)
	for _, w := range cr.Spec.Watch {
		for _, p := range w.Parsers {
			parserNameList[p.Name] = false
		}
		if w.Namespace == "" {
			v.ErrorMessage = v.ErrorMessage + "Watch section missing Namespace\n"
			v.Validated = false
		}

		if len(w.Parsers) == 0 {
			v.ErrorMessage = v.ErrorMessage + "Watch section missing Parsers\n"
			v.Validated = false
		}

		if len(w.Outputs) == 0 {
			v.ErrorMessage = v.ErrorMessage + "Watch section missing Outputs\n"
			v.Validated = false
		}
	}

	if len(cr.Spec.Parsers) == 0 {
		v.ErrorMessage = v.ErrorMessage + "Parsers definition missing\n"
		v.Validated = false
	}

	for _, p := range cr.Spec.Parsers {
		if _, present := parserNameList[p.Name]; present {
			parserNameList[p.Name] = true
		}
	}

	for k := range parserNameList {
		if !parserNameList[k] {
			v.ErrorMessage = v.ErrorMessage + "Parser - " + k + " definition not defined\n"
			v.Validated = false
		}
	}
}
