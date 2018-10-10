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

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

// default to 15 minute timeout
const DEFAULT_CMD_TIMEOUT_IN_SECONDS = 900

// IsFileExists probes whether the given pod's filesystem contains something
// at the indicated filepath. The returned boolean will be true if the file
// was found. If false, the returned error will be nil if the file is known to
// be missing, or non-nil if the probe failed to execute.
func IsFileExists(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	filePath string,
) (bool, error) {

	command := []string{"test", "-f", filePath}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdOut bytes.Buffer
	ioStreams := &streams{out: &stdOut}
	execErr := execCommand(cr, podName, command, ioStreams)
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

// CreateDir creates a directory (and any parent directors)
// in the filesystem of the given pod
func CreateDir(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	dirName string,
) error {

	command := []string{"mkdir", "-p", dirName}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdOut bytes.Buffer
	ioStreams := &streams{out: &stdOut}
	return execCommand(cr, podName, command, ioStreams)
}

// CreateFile takes the stream from the given reader, and writes it to the
// indicated filepath in the filesystem of the given pod.
func CreateFile(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	filePath string,
	reader io.Reader,
) error {

	createDirErr := CreateDir(cr, podName, filepath.Dir(filePath))

	if createDirErr != nil {
		return createDirErr
	}

	command := []string{"tee", filePath}
	ioStreams := &streams{
		in: reader,
	}
	shared.LogInfof(
		cr,
		"creating file{%s} in pod{%s}",
		filePath,
		podName,
	)
	execErr := execCommand(cr, podName, command, ioStreams)
	if execErr != nil {
		return execErr
	}
	return nil
}

// ReadFile takes the stream from the given writer, and writes to it the
// contents of the indicated filepath in the filesystem of the given pod.
// The returned boolean and error are interpreted in the same way as for
// IsFileExists.
func ReadFile(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	filePath string,
	writer io.Writer,
) (bool, error) {

	command := []string{"cat", filePath}
	ioStreams := &streams{
		out: writer,
	}
	shared.LogInfof(
		cr,
		"reading file{%s} in pod{%s}",
		filePath,
		podName,
	)
	execErr := execCommand(cr, podName, command, ioStreams)
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
	cr *kdv1.KubeDirectorCluster,
	podName string,
	description string,
	reader io.Reader,
) error {

	command := []string{execShell}
	ioStreams := &streams{
		in: reader,
	}
	shared.LogInfof(
		cr,
		"running %s in pod{%s}",
		description,
		podName,
	)
	execErr := execCommand(cr, podName, command, ioStreams)
	if execErr != nil {
		return execErr
	}
	return nil
}

// execCommand is a utility function for executing a command in a pod. It
// uses the given ioStreams to provide the command inputs and accept the
// command outputs. The command will be given DEFAULT_CMD_TIMEOUT_IN_SECONDS
// to complete.
func execCommand(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	command []string,
	ioStreams *streams,
) error {
	error, errch := execCommandAsync(
		cr,
		podName,
		command,
		ioStreams,
	)
	if error != nil {
		return error
	}

	var timeInMilliSeconds time.Duration = DEFAULT_CMD_TIMEOUT_IN_SECONDS * time.Second
	return execCommandWait(
		cr,
		podName,
		command,
		ioStreams,
		errch,
		timeInMilliSeconds,
	)
}

// execCommandAsync is a utility function for submitting a command in a pod.
// It uses the given ioStreams to provide the command inputs and accept the
// command outputs.
// The expectation is that the caller will invoke execCommandWait() to
// determine when the command has completed.
func execCommandAsync(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	command []string,
	ioStreams *streams,
) (error, chan error) {

	pod, podErr := observer.GetPod(cr.Namespace, podName)
	if podErr != nil {
		shared.LogErrorf(
			cr,
			"could not find pod{%s}: %v",
			podName,
			podErr,
		)
		return fmt.Errorf(
			"pod{%v} does not exist",
			podName,
		), nil
	}

	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return fmt.Errorf(
			"cannot connect to pod{%v} in phase %v",
			podName,
			pod.Status.Phase,
		), nil
	}

	foundContainer := false
	for _, container := range pod.Spec.Containers {
		if container.Name == appContainerName {
			foundContainer = true
			break
		}
	}
	if !foundContainer {
		return fmt.Errorf(
			"container{%s} does not exist in pod{%v}",
			appContainerName,
			podName,
		), nil
	}

	request := shared.Client.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(cr.Namespace).
		SubResource("exec").
		Param("container", appContainerName)
	request.VersionedParams(&v1.PodExecOptions{
		Container: appContainerName,
		Command:   command,
		Stdin:     ioStreams.in != nil,
		Stdout:    ioStreams.out != nil,
		Stderr:    ioStreams.errOut != nil,
	}, scheme.ParameterCodec)

	exec, initErr := remotecommand.NewSPDYExecutor(
		shared.Client.ClientConfig,
		"POST",
		request.URL(),
	)
	if initErr != nil {
		shared.LogErrorf(
			cr,
			"failed to init the executor: %v",
			initErr,
		)
		return errors.New("failed to initialize command executor"), nil
	}

	// Setup channel for remote command execution
	errch := make(chan error)

	go func() {
		execErr := exec.Stream(remotecommand.StreamOptions{
			Tty:    false,
			Stdin:  ioStreams.in,
			Stdout: ioStreams.out,
			Stderr: ioStreams.errOut,
		})
		if execErr != nil {
			errch <- execErr
			return
		}
		errch <- nil
		return
	}()

	return nil, errch
}

// execCommandWait is a utility function for waiting for the completion of a
// command submitted to a pod via the execCommandSync() function.
func execCommandWait(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	command []string,
	ioStreams *streams,
	errch chan error,
	timeInMilliSeconds time.Duration,
) error {
	timer := time.NewTimer(timeInMilliSeconds)
	defer timer.Stop()

	// Wait for command completion, or timeout
	select {
	case execErr := <-errch:
		close(errch)
		return execErr
	case <-timer.C:
		close(errch)
		return fmt.Errorf(
			"command{%v} sent to container{%s} in pod {%v} timed out in {%.3f} seconds",
			command,
			appContainerName,
			podName,
			timeInMilliSeconds.Seconds(),
		)
	}
}
