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
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// replicasPatchSpec is used to create PATCH operation input for modifying a
// statefulset's replicas count.
type replicasPatchSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int32  `json:"value"`
}

// defaultMountFolders identifies the set of member filesystems directories
// that will always be placed on shared persistent storage (when available).
var defaultMountFolders = []string{"/usr", "/opt", "/var", "/etc"}

// CreateStatefulSet creates in k8s a zero-replicas statefulset for
// implementing the given role.
func CreateStatefulSet(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) (*appsv1.StatefulSet, error) {

	statefulSet, err := getStatefulset(cr, role, 0)
	if err != nil {
		return nil, err
	}
	return statefulSet, sdk.Create(statefulSet)
}

// UpdateStatefulSetReplicas modifies an existing statefulset in k8s to have
// the given number of replicas.
func UpdateStatefulSetReplicas(
	cr *kdv1.KubeDirectorCluster,
	replicas int32,
	statefulSet *appsv1.StatefulSet,
) error {

	replicasPatch := []replicasPatchSpec{
		{
			Op:    "replace",
			Path:  "/spec/replicas",
			Value: replicas,
		},
	}
	replicasPatchBytes, patchErr := json.Marshal(replicasPatch)
	if patchErr == nil {
		patchErr = sdk.Patch(statefulSet, types.JSONPatchType, replicasPatchBytes)
	}

	return patchErr
}

// UpdateHeadlessService examines a current statefulset in k8s and may take
// steps to reconcile it to the desired spec, for properties other than the
// replicas count.
func UpdateStatefulSetNonReplicas(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	statefulSet *appsv1.StatefulSet,
) error {

	// If no spec, nothing to do.
	if role == nil {
		return nil
	}

	// TBD: We could compare the service against the expected service
	// (generated from the CR) and if there is a deviance in properties that
	// we need/expect to be under our control, other than the replicas
	// count, correct them here.

	return nil
}

// DeleteStatefulSet deletes a statefulset from k8s.
func DeleteStatefulSet(
	namespace string,
	statefulSetName string,
) error {

	toDelete := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: namespace,
		},
	}
	return sdk.Delete(toDelete)
}

// getStatefulset composes the spec for creating a statefulset in k8s, based
// on the given virtual cluster CR and for the purposes of implementing the
// given role.
func getStatefulset(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	replicas int32,
) (*appsv1.StatefulSet, error) {

	labels := labelsForRole(cr, role)
	startupScript := getStartupScript(cr)

	ports, portsErr := catalog.PortsForRole(cr, role.Name)
	if portsErr != nil {
		return nil, portsErr
	}

	var endpointPorts []v1.ContainerPort
	for _, port := range ports {
		containerPort := v1.ContainerPort{ContainerPort: port, Name: "port-" + strconv.Itoa(int(port))}
		endpointPorts = append(endpointPorts, containerPort)
	}

	// Check to see if app has requested additional directories to be persisted
	appPersistDirs, persistErr := catalog.AppPersistDirs(cr)
	if persistErr != nil {
		return nil, persistErr
	}

	// Create a combined unique list of directories that have be persisted
	// Start with default mounts
	persistDirs := make([]string, len(defaultMountFolders), len(defaultMountFolders)+len(appPersistDirs))
	copy(persistDirs, defaultMountFolders)

	// if the app directory is either same or a subdir of one of the default mount
	// dirs, we can skip them. if not we should add them to the persistDirs list
	for _, appDir := range appPersistDirs {
		isSubDir := false
		for _, defaultDir := range defaultMountFolders {
			// Get relative path of the app dir wrt defaultDir
			rel, _ := filepath.Rel(defaultDir, appDir)

			// If rel path doesn't start with "..", it is a subdir
			if !strings.HasPrefix(rel, "..") {
				shared.LogInfof(
					cr,
					"skipping {%s} from volume claim mounts. defaul dir {%s} covers it",
					appDir,
					defaultDir,
				)
				isSubDir = true
				break
			}
		}
		if !isSubDir {
			// Get the absolute path for the app dir
			abs, _ := filepath.Abs(appDir)

			persistDirs = append(persistDirs, abs)
		}
	}

	useServiceAccount := false
	volumeMounts, volumes, volumesErr := generateVolumeMounts(
		cr,
		role,
		persistDirs,
	)

	if volumesErr != nil {
		return nil, volumesErr
	}

	imageID, imageErr := catalog.ImageForRole(cr, role.Name)
	if imageErr != nil {
		return nil, imageErr
	}

	securityContext, securityErr := generateSecurityContext(cr)
	if securityErr != nil {
		return nil, securityErr
	}

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    cr.Name + "-" + role.Name + "-",
			Namespace:       cr.Namespace,
			OwnerReferences: ownerReferences(cr),
			Labels:          labels,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicas,
			ServiceName:         cr.Status.ClusterService,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					AutomountServiceAccountToken: &useServiceAccount,
					InitContainers:               getInitContainer(cr, role, pvcName, imageID, persistDirs),
					Containers: []v1.Container{
						{
							Name:            appContainerName,
							Image:           imageID,
							Resources:       role.Resources,
							Lifecycle:       &v1.Lifecycle{PostStart: &startupScript},
							Ports:           endpointPorts,
							VolumeMounts:    volumeMounts,
							SecurityContext: securityContext,
							Env:             role.EnvVars,
						},
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: getVolumeClaimTemplate(cr, role, pvcName),
		},
	}, nil
}

// getInitContainer prepares the init container spec to be used with the
// given role (for initializing the directory content placed on shared
// persistent storage). The result will be empty if the role does not use
// shared persistent storage.
func getInitContainer(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	pvcName string,
	imageID string,
	persistDirs []string,
) (initContainer []v1.Container) {

	if role.Storage.Size == "" {
		return
	}

	initVolumeMounts := generateInitVolumeMounts(pvcName)
	cpus, _ := resource.ParseQuantity("2")
	mem, _ := resource.ParseQuantity("512Mi")
	initContainer = []v1.Container{
		{
			Args: []string{
				"-c",
				generateInitContainerLaunch(persistDirs),
			},
			Command: []string{
				"/bin/bash",
			},
			Image: imageID,
			Name:  initContainerName,
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					"cpu":    cpus,
					"memory": mem,
				},
				Requests: v1.ResourceList{
					"cpu":    cpus,
					"memory": mem,
				},
			},
			VolumeMounts: initVolumeMounts,
		},
	}
	return
}

// getVolumeClaimTemplate prepares the PVC template to be used with the
// given role (for acquiring shared persistent storage). The result will be
// empty if the role does not use shared persistent storage.
func getVolumeClaimTemplate(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	pvcName string,
) (volTemplate []v1.PersistentVolumeClaim) {

	if role.Storage.Size == "" {
		return
	}

	volSize, _ := resource.ParseQuantity(role.Storage.Size)
	volTemplate = []v1.PersistentVolumeClaim{
		v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcName,
				Annotations: map[string]string{
					storageClassName: role.Storage.StorageClass,
				},
				OwnerReferences: ownerReferences(cr),
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{
					v1.ReadWriteOnce,
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: volSize,
					},
				},
			},
		},
	}
	return
}

// getStartupScript composes the startup script used for each app container.
// Currently this adds the virtual cluster's DNS subdomain to the resolv.conf
// search list.
func getStartupScript(
	cr *kdv1.KubeDirectorCluster,
) v1.Handler {

	return v1.Handler{
		Exec: &v1.ExecAction{
			Command: []string{
				"/bin/bash",
				"-c",
				"sed \"s/^search \\([^ ]\\+\\)/search " +
					cr.Status.ClusterService +
					".\\1 \\1/\" /etc/resolv.conf > /etc/resolv.conf.new;" +
					"cat /etc/resolv.conf.new > /etc/resolv.conf;" +
					"rm /etc/resolv.conf.new;" +
					"exit 0",
			},
		},
	}
}

// generateInitContainerLaunch generates the container entrypoint command for
// init containers. This command will populate the initial contents of the
// directories-to-be-persisted under the "/mnt" directory on the init
// container filesystem, then terminate the container.
func generateInitContainerLaunch(persistDirs []string) string {

	// To be safe in the case that this container is restarted by someone,
	// don't do this copy if the configmeta file already exists in /etc.
	launchCmd := "! [ -f /mnt" + configMetaFile + " ]" + " && " +
		"cp --parent -ax " + strings.Join(persistDirs, " ") + " /mnt || exit 0"

	return launchCmd
}

// generateVolumeMounts generates all of an app container's volume and mount
// specs for persistent storage, tmpfs, and systemctl support that are
// appropriate for members of the given role.
func generateVolumeMounts(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	persistDirs []string,
) ([]v1.VolumeMount, []v1.Volume, error) {

	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

	if role.Storage.Size != "" {
		volumeMounts = generateClaimMounts(pvcName, persistDirs)
	}

	tmpfsVolMnts, tmpfsVols := generateTmpfsSupport(cr)
	volumeMounts = append(volumeMounts, tmpfsVolMnts...)
	volumes = append(volumes, tmpfsVols...)

	isSystemdReqd, err := catalog.SystemdRequired(cr)

	if err != nil {
		return volumeMounts, volumes, err
	}

	if isSystemdReqd {
		cgroupVolMnts, cgroupVols := generateSystemdSupport(cr)
		volumeMounts = append(volumeMounts, cgroupVolMnts...)
		volumes = append(volumes, cgroupVols...)
	}

	return volumeMounts, volumes, nil
}

// generateClaimMounts creates the mount specs for all directories that are
// to be mounted from a persistent volume by an app container.
func generateClaimMounts(
	pvcName string,
	persistDirs []string,
) []v1.VolumeMount {

	var volumeMounts []v1.VolumeMount
	for _, folder := range persistDirs {
		volumeMount := v1.VolumeMount{
			MountPath: folder,
			Name:      pvcName,
			ReadOnly:  false,
			SubPath:   folder[1:],
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}
	return volumeMounts
}

// generateInitVolumeMounts creates the spec for mounting a persistent volume
// into an init container.
func generateInitVolumeMounts(
	pvcName string,
) []v1.VolumeMount {

	return []v1.VolumeMount{
		v1.VolumeMount{
			MountPath: "/mnt",
			Name:      pvcName,
			ReadOnly:  false,
		},
	}
}

// generateSystemdSupport creates the volume and mount specs necessary for
// supporting the use of systemd within an app container by mounting
// appropriate /sys/fs/cgroup directories from the host.
func generateSystemdSupport(
	cr *kdv1.KubeDirectorCluster,
) ([]v1.VolumeMount, []v1.Volume) {

	cgroupFsName := cr.Name + "-cgroupfs"
	systemdFsName := cr.Name + "-systemd"
	volumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:      cgroupFsName,
			MountPath: cgroupFsVolume,
			ReadOnly:  true,
		},
		v1.VolumeMount{
			Name:      systemdFsName,
			MountPath: systemdFsVolume,
		},
	}
	volumes := []v1.Volume{
		v1.Volume{
			Name: cgroupFsName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: cgroupFsVolume,
				},
			},
		},
		v1.Volume{
			Name: systemdFsName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: systemdFsVolume,
				},
			},
		},
	}
	return volumeMounts, volumes
}

// generateTmpfsSupport creates the volume and mount specs necessary for
// backing an app container's /tmp and /run directories with a ramdisk. Limit
// the size of the ramdisk to tmpFsVolSize.
func generateTmpfsSupport(
	cr *kdv1.KubeDirectorCluster,
) ([]v1.VolumeMount, []v1.Volume) {

	volumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "tmpfs",
			MountPath: "/tmp",
		},
		v1.VolumeMount{
			Name:      "tmpfs",
			MountPath: "/run",
		},
	}
	maxTmpSize, _ := resource.ParseQuantity(tmpFsVolSize)
	volumes := []v1.Volume{
		v1.Volume{
			Name: "tmpfs",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: &maxTmpSize,
				},
			},
		},
	}
	return volumeMounts, volumes
}

// generateSecurityContext creates security context with Add Capabilities property
// based on app's capability list. If app doesn't require additional capabilities
// return nil
func generateSecurityContext(
	cr *kdv1.KubeDirectorCluster,
) (*v1.SecurityContext, error) {

	appCapabilities, err := catalog.AppCapabilities(cr)
	if err != nil {
		return nil, err
	}

	if len(appCapabilities) == 0 {
		return nil, err
	}

	return &v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Add: appCapabilities,
		},
	}, nil
}
