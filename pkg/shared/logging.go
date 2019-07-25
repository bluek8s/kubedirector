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
	"fmt"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

// LogInfo logs the given message at Info level.
func LogInfo(
	logger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	eventReason string,
	msg string,
) {

	logger.Info(msg)

	if eventReason != "" {
		LogEvent(
			cr,
			v1.EventTypeNormal,
			eventReason,
			msg,
		)
	}
}

// LogInfof logs the given message format and payload at Info level.
func LogInfof(
	logger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	eventReason string,
	format string,
	args ...interface{},
) {

	logger.Info(fmt.Sprint(format, args))

	if eventReason != EventReasonNoEvent {
		LogEventf(
			cr,
			v1.EventTypeNormal,
			eventReason,
			format,
			args...,
		)
	}
}

// LogError logs the given message at Error level.
func LogError(
	logger logr.Logger,
	err error,
	cr *kdv1.KubeDirectorCluster,
	eventReason string,
	msg string,
) {

	logger.Error(err, msg)

	if eventReason != EventReasonNoEvent {
		LogEvent(
			cr,
			v1.EventTypeWarning,
			eventReason,
			msg,
		)
	}
}

// LogErrorf logs the given message format and payload at Error level.
func LogErrorf(
	logger logr.Logger,
	err error,
	cr *kdv1.KubeDirectorCluster,
	eventReason string,
	format string,
	args ...interface{},
) {

	logger.Error(err, fmt.Sprint(format, args))

	if eventReason != EventReasonNoEvent {
		LogEventf(
			cr,
			v1.EventTypeWarning,
			eventReason,
			format,
			args...,
		)
	}
}

// LogEvent posts an event to event recorder with the given msg using the
// CR object as reference
func LogEvent(
	cr *kdv1.KubeDirectorCluster,
	eventType string,
	eventReason string,
	msg string,
) {

	LogEventf(
		cr,
		eventType,
		eventReason,
		msg,
	)
}

// LogEventf posts an event to event recorder with the given message format
// and payload using the CR object as reference
func LogEventf(
	cr *kdv1.KubeDirectorCluster,
	eventType string,
	eventReason string,
	format string,
	args ...interface{},
) {

	ref, _ := reference.GetReference(scheme.Scheme, cr)

	eventRecorder.Eventf(
		ref,
		eventType,
		eventReason,
		format,
		args...,
	)
}
