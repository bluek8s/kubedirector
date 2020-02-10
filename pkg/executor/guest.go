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
// be missing, or non-nil if the probe failed to execute.
func IsFileExists(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	filePath string,
) (bool, error) {

	command := []string{"test", "-f", filePath}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdOut bytes.Buffer
	ioStreams := &Streams{Out: &stdOut}
	execErr := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		expectedContainerID,
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
				return false, nil
			}
		}
		// Some error, other than file does not exist, occured.
		return false, execErr
	}
	// The file exists.
	return true, nil
}

// CreateDir creates a directory (and any parent directors) in the filesystem
// of the given pod.
func CreateDir(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	dirName string,
) error {

	command := []string{"mkdir", "-p", dirName}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdErr bytes.Buffer
	ioStreams := &Streams{ErrOut: &stdErr}
	err := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		expectedContainerID,
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
	return err
}

// RemoveDir removes a directory.
func RemoveDir(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	dirName string,
	ignoreNotEmpty bool,
) error {

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
	err := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		expectedContainerID,
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
	return err
}

// CreateFile takes the stream from the given reader, and writes it to the
// indicated filepath in the filesystem of the given pod.
func CreateFile(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	filePath string,
	reader io.Reader,
) error {

	createDirErr := CreateDir(
		reqLogger,
		obj,
		namespace,
		podName,
		expectedContainerID,
		containerName,
		filepath.Dir(filePath),
	)
	if createDirErr != nil {
		return createDirErr
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
		expectedContainerID,
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
	expectedContainerID string,
	containerName string,
	filePath string,
	writer io.Writer,
) (bool, error) {

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
	execErr := ExecCommand(
		reqLogger,
		obj,
		namespace,
		podName,
		expectedContainerID,
		containerName,
		command,
		ioStreams,
	)
	if execErr != nil {
		coe, iscoe := execErr.(exec.CodeExitError)
		if iscoe {
			if coe.ExitStatus() == 1 {
				return false, nil
			}
		}
		return false, execErr
	}
	return true, nil
}

// RunScript takes the stream from the given reader, and executes it as a
// shell script in the given pod.
func RunScript(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	description string,
	reader io.Reader,
) error {

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
		expectedContainerID,
		containerName,
		command,
		ioStreams,
	)
}

// ExecCommand is a utility function for executing a command in a pod. It
// uses the given ioStreams to provide the command inputs and accept the
// command outputs.
func ExecCommand(
	reqLogger logr.Logger,
	obj runtime.Object,
	namespace string,
	podName string,
	expectedContainerID string,
	containerName string,
	command []string,
	ioStreams *Streams,
) error {

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
		return fmt.Errorf(
			"pod{%v} does not exist",
			podName,
		)
	}

	foundContainer := false
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == containerName {
			foundContainer = true
			if containerStatus.ContainerID != expectedContainerID {
				return errors.New("container ID changed during configuration")
			}
			break
		}
	}
	if !foundContainer {
		return fmt.Errorf(
			"container{%s} does not exist in pod{%v}",
			containerName,
			podName,
		)
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf(
			"cannot connect to pod{%v} in phase %v",
			podName,
			pod.Status.Phase,
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
		return errors.New("failed to initialize command executor")
	}
	execErr := exec.Stream(remotecommand.StreamOptions{
		Tty:    false,
		Stdin:  ioStreams.In,
		Stdout: ioStreams.Out,
		Stderr: ioStreams.ErrOut,
	})

	return execErr
}
