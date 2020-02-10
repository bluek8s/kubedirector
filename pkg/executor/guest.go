// Copyright 2019 Hewlett Packard Enterprise Development LP

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

// IsFileExists probes whether the given pod's filesystem contains something
// at the indicated filepath. The returned boolean will be true if the file
// was found. If false, the returned error will be nil if the file is known to
// be missing, or non-nil if the probe failed to execute. The returned string
// pointer is the container ID if successfully found.
func IsFileExists(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	filePath string,
) (*string, bool, error) {

	command := []string{"test", "-f", filePath}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdOut bytes.Buffer
	ioStreams := &Streams{Out: &stdOut}
	containerID, execErr := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
	if execErr != nil {
		// Determine which type of error occured
		coe, iscoe := execErr.(exec.CodeExitError)
		if iscoe {
			// If the command failed with a CodeExitError error and an exit
			// code of 1, this means that the file existence check completed
			// successfully, but the file does not exist.
			if coe.ExitStatus() == 1 {
				return containerID, false, nil
			}
		}
		// Some error, other than file does not exist, occured.
		return containerID, false, execErr
	}
	// The file exists.
	return containerID, true, nil
}

// CreateDir creates a directory (and any parent directors) in the filesystem
// of the given pod. The return value is a pointer to the container ID (if
// successfully found) and any error.
func CreateDir(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	dirName string,
) (*string, error) {

	command := []string{"mkdir", "-p", dirName}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdErr bytes.Buffer
	ioStreams := &Streams{ErrOut: &stdErr}
	containerID, err := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
	if err != nil {
		err = fmt.Errorf("mkdir failed: %s\n%s",
			stdErr.String(),
			err.Error(),
		)
	}
	return containerID, err
}

// RemoveDir removes a directory. The return value is a pointer to the
// container ID (if successfully found) and any error.
func RemoveDir(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	dirName string,
	ignoreNotEmpty bool,
) (*string, error) {

	var command []string
	if ignoreNotEmpty {
		command = []string{"rmdir", "--ignore-fail-on-non-empty", dirName}
	} else {
		command = []string{"rmdir", dirName}
	}

	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdErr bytes.Buffer
	ioStreams := &Streams{ErrOut: &stdErr}
	containerID, err := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
	if err != nil {
		errStr := stdErr.String()
		if strings.Contains(errStr, "No such file or directory") {
			err = nil
		} else {
			err = fmt.Errorf("rmdir failed: %s", errStr)
		}
	}
	return containerID, err
}

// CreateFile takes the stream from the given reader, and writes it to the
// indicated filepath in the filesystem of the given pod. The return value is
// a pointer to the container ID (if successfully found) and any error.
func CreateFile(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	filePath string,
	reader io.Reader,
) (*string, error) {

	containerID, createDirErr := CreateDir(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		filepath.Dir(filePath),
	)
	if createDirErr != nil {
		return containerID, createDirErr
	}

	command := []string{"tee", filePath}
	ioStreams := &Streams{
		In: reader,
	}
	shared.LogInfof(
		reqLogger,
		obj,
		shared.EventReasonNoEvent,
		"creating file{%s} in pod{%s}",
		filePath,
		podName,
	)
	return ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
}

// ReadFile takes the stream from the given writer, and writes to it the
// contents of the indicated filepath in the filesystem of the given pod.
// The returned boolean and error are interpreted in the same way as for
// IsFileExists.
func ReadFile(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	filePath string,
	writer io.Writer,
) (*string, bool, error) {

	command := []string{"cat", filePath}
	ioStreams := &Streams{
		Out: writer,
	}
	shared.LogInfof(
		reqLogger,
		obj,
		shared.EventReasonNoEvent,
		"reading file{%s} in pod{%s}",
		filePath,
		podName,
	)
	containerID, execErr := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
	if execErr != nil {
		coe, iscoe := execErr.(exec.CodeExitError)
		if iscoe {
			if coe.ExitStatus() == 1 {
				return containerID, false, nil
			}
		}
		return containerID, false, execErr
	}
	return containerID, true, nil
}

// RunScript takes the stream from the given reader, and executes it as a
// shell script in the given pod. The return value is a pointer to the
// container ID (if successfully found) and any error.
func RunScript(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	description string,
	reader io.Reader,
) (*string, error) {

	command := []string{execShell}
	ioStreams := &Streams{
		In: reader,
	}
	shared.LogInfof(
		reqLogger,
		obj,
		shared.EventReasonNoEvent,
		"running %s in pod{%s}",
		description,
		podName,
	)
	return ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		containerName,
		command,
		ioStreams,
	)
}

// ExecCommand is a utility function for executing a command in a pod. It
// uses the given ioStreams to provide the command inputs and accept the
// command outputs. The return value is a pointer to the container ID (if
// successfully found) and any error.
func ExecCommand(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	containerName string,
	command []string,
	ioStreams *Streams,
) (*string, error) {

	var foundContainerID *string

	pod, podErr := observer.GetPod(namespace, podName)
	if podErr != nil {
		shared.LogErrorf(
			reqLogger,
			podErr,
			obj,
			shared.EventReasonNoEvent,
			"could not find pod{%s}",
			podName,
		)
		return foundContainerID, fmt.Errorf(
			"pod{%v} does not exist",
			podName,
		)
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == containerName {
			foundContainerID = &containerStatus.ContainerID
		}
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return foundContainerID, fmt.Errorf(
			"cannot connect to pod{%v} in phase %v",
			podName,
			pod.Status.Phase,
		)
	}

	foundContainer := false
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			foundContainer = true
			break
		}
	}
	if !foundContainer {
		return foundContainerID, fmt.Errorf(
			"container{%s} does not exist in pod{%v}",
			containerName,
			podName,
		)
	}

	request := shared.ClientSet().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName)
	request.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     ioStreams.In != nil,
		Stdout:    ioStreams.Out != nil,
		Stderr:    ioStreams.ErrOut != nil,
	}, scheme.ParameterCodec)

	exec, initErr := remotecommand.NewSPDYExecutor(
		shared.Config(),
		"POST",
		request.URL(),
	)
	if initErr != nil {
		shared.LogError(
			reqLogger,
			initErr,
			obj,
			shared.EventReasonNoEvent,
			"failed to init the executor",
		)
		return foundContainerID, errors.New("failed to initialize command executor")
	}
	execErr := exec.Stream(remotecommand.StreamOptions{
		Tty:    false,
		Stdin:  ioStreams.In,
		Stdout: ioStreams.Out,
		Stderr: ioStreams.ErrOut,
	})

	return foundContainerID, execErr
}
