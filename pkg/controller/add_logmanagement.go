package controller

import (
	"github.com/log_management/logging-operator/pkg/controller/logmanagement"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, logmanagement.Add)
}
