package controller

import (
	"github.com/bluek8s/kubedirector/pkg/controller/kubedirectorcluster"
)

func init() {

	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kubedirectorcluster.Add)
}
