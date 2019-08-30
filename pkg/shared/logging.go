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

	"github.com/go-logr/logr"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

// LogInfo logs the given message at Info level.
func LogInfo(
	logger logr.Logger,
	obj runtime.Object,
	eventReason string,
	msg string,
) {

	logger.Info(msg)

	if eventReason != "" {
		LogEvent(
			obj,
			v1.EventTypeNormal,
			eventReason,
			msg,
		)
	}
}

// LogInfof logs the given message format and payload at Info level.
func LogInfof(
	logger logr.Logger,
	obj runtime.Object,
	eventReason string,
	format string,
	args ...interface{},
) {

	logger.Info(fmt.Sprintf(format, args...))

	if eventReason != EventReasonNoEvent {
		LogEventf(
			obj,
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
	obj runtime.Object,
	eventReason string,
	msg string,
) {

	logger.Error(err, msg)

	if eventReason != EventReasonNoEvent {
		LogEvent(
			obj,
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
	obj runtime.Object,
	eventReason string,
	format string,
	args ...interface{},
) {

	logger.Error(err, fmt.Sprintf(format, args...))

	if eventReason != EventReasonNoEvent {
		LogEventf(
			obj,
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
	obj runtime.Object,
	eventType string,
	eventReason string,
	msg string,
) {

	LogEventf(
		obj,
		eventType,
		eventReason,
		msg,
	)
}

// LogEventf posts an event to event recorder with the given message format
// and payload using the CR object as reference
func LogEventf(
	obj runtime.Object,
	eventType string,
	eventReason string,
	format string,
	args ...interface{},
) {

	ref, _ := reference.GetReference(scheme.Scheme, obj)

	eventRecorder.Eventf(
		ref,
		eventType,
		eventReason,
		format,
		args...,
	)
}
