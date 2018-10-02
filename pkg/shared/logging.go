// Copyright 2018 BlueData Software, Inc.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shared

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/sirupsen/logrus"
)

const (
	msgPrefix = "cluster{%s/%s}: "
)

// appendArgs is an internal utility function. It adds the CR namespace and
// name to the beginning of the given arg list and returns the resulting list.
func appendArgs(
	cr *kdv1.KubeDirectorCluster,
	args ...interface{},
) []interface{} {

	newArgs := make([]interface{}, 2, 2+len(args))
	newArgs[0] = cr.Namespace
	newArgs[1] = cr.Name
	return append(newArgs, args...)
}

// LogInfo logs the given message at Info level.
func LogInfo(
	cr *kdv1.KubeDirectorCluster,
	msg string,
) {

	logrus.Infof(
		msgPrefix+msg,
		cr.Namespace,
		cr.Name,
	)
}

// LogInfof logs the given message format and payload at Info level.
func LogInfof(
	cr *kdv1.KubeDirectorCluster,
	format string,
	args ...interface{},
) {

	logrus.Infof(
		msgPrefix+format,
		appendArgs(cr, args...)...,
	)
}

// LogWarn logs the given message at Warning level.
func LogWarn(
	cr *kdv1.KubeDirectorCluster,
	msg string,
) {

	logrus.Warnf(
		msgPrefix+msg,
		cr.Namespace,
		cr.Name,
	)
}

// LogWarnf logs the given message format and payload at Warning level.
func LogWarnf(
	cr *kdv1.KubeDirectorCluster,
	format string,
	args ...interface{},
) {

	logrus.Warnf(
		msgPrefix+format,
		appendArgs(cr, args...)...,
	)
}

// LogError logs the given message at Error level.
func LogError(
	cr *kdv1.KubeDirectorCluster,
	msg string,
) {

	logrus.Errorf(
		msgPrefix+msg,
		cr.Namespace,
		cr.Name,
	)
}

// LogErrorf logs the given message format and payload at Error level.
func LogErrorf(
	cr *kdv1.KubeDirectorCluster,
	format string,
	args ...interface{},
) {

	logrus.Errorf(
		msgPrefix+format,
		appendArgs(cr, args...)...,
	)
}
