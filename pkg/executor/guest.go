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
        "strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// IsFileExists probes whether the given pod's filesystem contains something
// at the indicated filepath.
func IsFileExists(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	filePath string,
) (bool, error) {

	var (
		stdOut bytes.Buffer
		stdErr bytes.Buffer
	)
	command := []string{"test", "-f", filePath}
	ioStreams := &streams{
		out:    &stdOut,
		errOut: &stdErr,
	}
	execErr := execCommand(cr, podName, command, ioStreams)
        if  execErr != nil {

shared.LogErrorf(
cr,
"command{%s} IsFileExists FAILED: %v",
command,
execErr,
)


                // If the command fails with the error "command terminated with exit code 1",
    		// this means the file existence check completed successfully, but the file does not exist.
                // Otherwise the command failed for some other reason.
		if strings.Compare(strings.TrimRight(execErr.Error(), "\r\n") , "command terminated with exit code 1") == 0 {

shared.LogErrorf(
cr,
"command{%s} IsFileExists FAILED with exit code match - returning nil for error",
command,
)


                   return false, nil
                }

shared.LogErrorf(
cr,
"command{%s} IsFileExists FAILED WITHOUT exit code match - returning NON nil for error",
command,
)
                return false, execErr
        }
        return true, nil
}

// CreateDir creates a directory (and any parent directors)
// in the filesystem of the given pod
func CreateDir(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	dirName string,
) error {

	var (
		stdOut bytes.Buffer
		stdErr bytes.Buffer
	)
	command := []string{"mkdir", "-p", dirName}
	ioStreams := &streams{
		out:    &stdOut,
		errOut: &stdErr,
	}
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
// command outputs.
func execCommand(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	command []string,
	ioStreams *streams,
) error {

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
		)
	}

	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return fmt.Errorf(
			"cannot connect to pod{%v} in phase %v",
			podName,
			pod.Status.Phase,
		)
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
		)
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
		return errors.New("failed to initialize command executor")
	}
	execErr := exec.Stream(remotecommand.StreamOptions{
		Tty:    false,
		Stdin:  ioStreams.in,
		Stdout: ioStreams.out,
		Stderr: ioStreams.errOut,
	})

	return execErr
}
