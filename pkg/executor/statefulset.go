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
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/shared"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultMountFolders identifies the set of member filesystems directories
// that will always be placed on shared persistent storage (when available).
var defaultMountFolders = []string{"/etc"}

// appConfigDefaultMountFolders identifies set of member filesystems directories
// that will always be placed on shared persistent storage, if app config is provided
// for a role
var appConfigDefaultMountFolders = []string{"/etc", "/opt", "/usr"}

// CreateStatefulSet creates in k8s a zero-replicas statefulset for
// implementing the given role.
func CreateStatefulSet(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	nativeSystemdSupport bool,
	role *kdv1.Role,
	roleStatus *kdv1.RoleStatus,
) (*appsv1.StatefulSet, error) {

	statefulSet, err := getStatefulset(
		reqLogger,
		cr,
		nativeSystemdSupport,
		role,
		roleStatus,
		0,
	)
	if err != nil {
		return nil, err
	}
	return statefulSet, shared.Create(context.TODO(), statefulSet)
}

// UpdateStatefulSetReplicas modifies an existing statefulset in k8s to have
// the given number of replicas.
func UpdateStatefulSetReplicas(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	replicas int32,
	statefulSet *appsv1.StatefulSet,
) error {

	*statefulSet.Spec.Replicas = replicas
	err := shared.Update(context.TODO(), statefulSet)
	if err == nil {
		return nil
	}

	// See https://github.com/bluek8s/kubedirector/issues/194
	// Migrate Client().Update() calls back to Patch() calls.

	if !errors.IsConflict(err) {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonNoEvent,
			"failed to update statefulset",
		)
		return err
	}

	// If there was a resourceVersion conflict then fetch a more
	// recent version of the statefulset and attempt to update that.
	name := types.NamespacedName{
		Namespace: statefulSet.Namespace,
		Name:      statefulSet.Name,
	}
	*statefulSet = appsv1.StatefulSet{}
	err = shared.Get(context.TODO(), name, statefulSet)
	if err != nil {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonNoEvent,
			"failed to retrieve statefulset",
		)
		return err
	}

	*statefulSet.Spec.Replicas = replicas
	err = shared.Update(context.TODO(), statefulSet)
	if err != nil {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonNoEvent,
			"failed to update statefulset",
		)
	}
	return err
}

// UpdateStatefulSetNonReplicas examines a current statefulset in k8s and may take
// steps to reconcile it to the desired spec, for properties other than the
// replicas count.
func UpdateStatefulSetNonReplicas(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	statefulSet *appsv1.StatefulSet,
	clusterIsReady bool,
) error {

	// If no spec, nothing to do.
	if role == nil {
		return nil
	}

	// We could compare the statefulset against the expected statefulset
	// (generated from the CR) and if there is a deviance in properties that we
	// need/expect to be under our control, other than the replicas count,
	// correct them here.

	patchedRes := *statefulSet
	needPatch := false

	// Check the owner reference.
	if !shared.OwnerReferencesPresent(cr, statefulSet.OwnerReferences) {
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonNoEvent,
			"repairing owner ref on statefulset{%s}",
			statefulSet.Name,
		)

		// So, what to do. Do we add our owner ref to the existing ones? What if
		// something else is claiming to be controller? Probably some stale ref
		// left by a bad backup/restore process? We're just going to nuke any
		// existing owner refs.
		patchedRes.OwnerReferences = shared.OwnerReferences(cr)
		needPatch = true
	}

	// https://github.com/bluek8s/kubedirector/issues/229
	// We should also compare the current statefulset container image against
	// the image defined for the corresponding role in the KDCluster spec. If images
	// are not the same, KDCluster spec image has a higher priority, so we should
	// upgrade the statefulset one.
	// Make sure, that according this logic, there is no sence to edit a statefulset
	// directly, as it will be reconciled back to the KDCluster spec state.

	containers := shared.StatefulSetContainers(statefulSet)
	currentRoleImage := containers[0].Image

	appRoleImage, err := catalog.ImageForRole(cr, role.Name)
	if err != nil {
		return err
	}

	// There may appear the situation when during image upgrade we set to the
	// cluster spec an incorrect image tag or image cannot be pulled from a repo by some
	// other reason. It may cause that a member of the role falls
	// into an endless restart loop getting ErrImagePull or ImagePullBackOff errors.
	// Than, the cluster falls into non-ready state and this role status will be always
	// in upgrading state and reject any try to re-upgrade an image.
	// For avoiding this situation we should have an ability to re-edit cluster spec
	// and give a chance to the role status to restart again.
	// A rollback process will start when an cluster spec role image will be equal to a
	// last image which was sucessfully performed on the role pods
	rollback := false

	// Check, if cluster is not ready and current role status has upgrading members
	roleStatusIsUpgrading := shared.RoleStatusIsUpgrading(cr, role.Name)

	if !clusterIsReady && roleStatusIsUpgrading {
		if err != nil {
			return err
		}
		rbKey := cr.Name + cr.Namespace
		rbInfo := shared.ClusterRollbackInfoMap[rbKey]

		if cr.AppSpec == nil {
			_, err := catalog.GetApp(cr)
			if err != nil {
				return err
			}
		}

		// Check if rollback info for this cluster (the last worked app descriptor)
		// matches with the current app spec. If so, permit rollback
		if rbInfo != nil && cr.AppSpec != nil {
			rollback = rbInfo.AppID == cr.Spec.AppID &&
				rbInfo.DistroID == cr.AppSpec.Spec.DistroID &&
				rbInfo.Version == cr.AppSpec.Spec.Version

			if rollback {
				rs, err := shared.GetRoleStatusByName(cr, role.Name)
				if err != nil {
					return err
				}
				(*rs).UpgradingMembers = nil
			}

		}
	}

	// Check is KDCluster in configured state and
	// is the roleStatus currently not upgrading
	readyToUpgrade := clusterIsReady && !roleStatusIsUpgrading

	// Or, check do we need to rollback a statefulset image
	if readyToUpgrade || rollback {

		if appRoleImage != currentRoleImage {
			patchedRes.Spec.Template.Spec.Containers = make([]v1.Container, len(containers))
			patchedContainers := shared.StatefulSetContainers(&patchedRes)
			copy(patchedContainers, containers)
			patchedContainers[0].Image = appRoleImage

			// Set current role members count should be upgraded
			// It will be used at the syncMembers() step
			rs, err := shared.GetRoleStatusByName(cr, role.Name)
			if err != nil {
				return err
			}

			if (*rs).UpgradingMembers == nil {
				(*rs).UpgradingMembers = make(map[string]*string)
			}

			for _, m := range (*rs).Members {
				// This check added for restart case.
				// If proposed clusterRoleImage is the same as already run in member container
				// we shouldn't add this member to UpgradeMembers list.
				if m.ContainerImage != appRoleImage {
					(*rs).UpgradingMembers[m.Pod] = &appRoleImage
				}
			}

			needPatch = true
		}
	}

	if !needPatch {
		return nil
	}

	patchErr := shared.Patch(
		context.TODO(),
		statefulSet,
		&patchedRes,
	)
	return patchErr
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
	return shared.Delete(context.TODO(), toDelete)
}

// getStatefulset composes the spec for creating a statefulset in k8s, based
// on the given virtual cluster CR and for the purposes of implementing the
// given role.
func getStatefulset(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	nativeSystemdSupport bool,
	role *kdv1.Role,
	roleStatus *kdv1.RoleStatus,
	replicas int32,
) (*appsv1.StatefulSet, error) {

	labels := labelsForStatefulSet(cr, role)
	podLabels := labelsForPod(cr, role)
	annotations := annotationsForStatefulSet(cr, role)
	podAnnotations := annotationsForPod(cr, role)
	startupScript := getStartupScript(cr)

	portInfoList, portsErr := catalog.PortsForRole(cr, role.Name)
	if portsErr != nil {
		return nil, portsErr
	}

	var endpointPorts []v1.ContainerPort
	for _, portInfo := range portInfoList {
		containerPort := v1.ContainerPort{
			ContainerPort: portInfo.Port,
			Name:          portInfo.ID,
		}
		endpointPorts = append(endpointPorts, containerPort)
	}

	// Check to see if app has requested additional directories to be persisted
	appPersistDirs, persistErr := catalog.AppPersistDirs(cr, role.Name)
	if persistErr != nil {
		return nil, persistErr
	}

	defaultPersistDirs := &defaultMountFolders

	// Check if there is an app config package for this role, If so we have
	// to add additional defaults
	setupURL, setupURLErr := catalog.AppSetupPackageURL(cr, role.Name)
	if setupURLErr != nil {
		return nil, setupURLErr
	}

	if setupURL != "" {
		defaultPersistDirs = &appConfigDefaultMountFolders
	}

	// Create a combined unique list of directories that have be persisted
	// Start with default mounts
	var maxLen = len(*defaultPersistDirs)
	if appPersistDirs != nil {
		maxLen += len(*appPersistDirs)
	}
	persistDirs := make([]string, len(*defaultPersistDirs), maxLen)
	copy(persistDirs, *defaultPersistDirs)

	// if the app directory is either same or a subdir of one of the default mount
	// dirs, we can skip them. if not we should add them to the persistDirs list
	// Also eliminate any duplicates or sub-dirs from appPersistDirs as well
	if appPersistDirs != nil {
		for _, appDir := range *appPersistDirs {
			isSubDir := false
			for _, defaultDir := range persistDirs {
				// Get relative path of the app dir wrt defaultDir
				rel, _ := filepath.Rel(defaultDir, appDir)

				// If rel path doesn't start with "..", it is a subdir
				if !strings.HasPrefix(rel, "..") {
					shared.LogInfof(
						reqLogger,
						cr,
						shared.EventReasonNoEvent,
						"skipping {%s} from volume claim mounts. dir {%s} covers it",
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
	}

	useServiceAccount := false
	if role.ServiceAccountName != "" {
		useServiceAccount = true
	}
	volumeMounts, volumes, volumesErr := generateVolumeMounts(
		cr,
		role,
		PvcNamePrefix,
		nativeSystemdSupport,
		persistDirs,
	)

	if volumesErr != nil {
		return nil, volumesErr
	}

	// check if BlockStorage field is present. If it is, create a volumeDevices field
	var volumeDevices []v1.VolumeDevice
	if role.BlockStorage != nil {

		numDevices := *role.BlockStorage.NumDevices

		for i := int32(0); i < numDevices; i++ {

			deviceID := strconv.FormatInt(int64(i), 10)
			devicePath := *role.BlockStorage.Path + deviceID
			deviceName := blockPvcNamePrefix + deviceID

			volumeDevice := v1.VolumeDevice{
				Name:       deviceName,
				DevicePath: devicePath,
			}
			volumeDevices = append(volumeDevices, volumeDevice)

		}

	}
	imageID, imageErr := catalog.ImageForRole(cr, role.Name)
	if imageErr != nil {
		return nil, imageErr
	}

	securityContext, securityErr := generateSecurityContext(cr)
	if securityErr != nil {
		return nil, securityErr
	}

	vct := getVolumeClaimTemplate(cr, role, PvcNamePrefix)

	sset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       cr.Namespace,
			OwnerReferences: shared.OwnerReferences(cr),
			Labels:          labels,
			Annotations:     annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicas,
			ServiceName:         cr.Status.ClusterService,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: v1.PodSpec{
					AutomountServiceAccountToken: &useServiceAccount,
					InitContainers: getInitContainer(
						cr,
						role,
						PvcNamePrefix,
						imageID,
						persistDirs,
					),
					Affinity:           role.Affinity,
					ServiceAccountName: role.ServiceAccountName,
					Containers: []v1.Container{
						{
							Name:            AppContainerName,
							Image:           imageID,
							Resources:       role.Resources,
							Lifecycle:       &v1.Lifecycle{PostStart: &startupScript},
							Ports:           endpointPorts,
							VolumeMounts:    volumeMounts,
							VolumeDevices:   volumeDevices,
							SecurityContext: securityContext,
							Env:             chkModifyEnvVars(role),
							TTY:             hasTTY(cr, role.Name),
							Stdin:           hasSTDIN(cr, role.Name),
						},
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: vct,
		},
	}

	namingScheme := *cr.Spec.NamingScheme
	if (roleStatus == nil) || (roleStatus.StatefulSet == "") {
		if namingScheme == v1beta1.CrNameRole {
			sset.ObjectMeta.GenerateName = MungObjectName(cr.Name + "-" + role.Name)
			sset.ObjectMeta.GenerateName += "-"
		} else if namingScheme == v1beta1.UID {
			sset.ObjectMeta.GenerateName = statefulSetNamePrefix
		}
	} else {
		sset.ObjectMeta.Name = roleStatus.StatefulSet
	}

	return sset, nil
}

// chkModifyEnvVars checks a role's resource requests. If an NVIDIA GPU resource has
// NOT been requested for the role, a work-around is added (as an environment variable), to
// avoid a GPU being surfaced anyway in a container related to the role
func chkModifyEnvVars(
	role *kdv1.Role,
) (envVar []v1.EnvVar) {

	envVar = role.EnvVars
	rsrcmap := role.Resources.Requests
	// return the role's environment variables unmodified, if an NVIDIA GPU is
	// indeed a resource requested for this role
	if quantity, found := rsrcmap[nvidiaGpuResourceName]; found == true && quantity.IsZero() != true {
		return envVar
	}

	// add an environment variable, as a work-around to ensure that an NVIDIA GPU is
	// not visible in a container (related to this role) for which an NVIDIA GPU resource
	// has not been requested (or the key for the NVIDIA GPU resource has been specified, but
	// with a quantity of zero)
	envVarToAdd := v1.EnvVar{
		Name:  nvidiaGpuVisWorkaroundEnvVarName,
		Value: nvidiaGpuVisWorkaroundEnvVarValue,
		// ValueFrom not used
	}
	envVar = append(envVar, envVarToAdd)
	return
}

// getInitContainer prepares the init container spec to be used with the
// given role (for initializing the directory content placed on shared
// persistent storage). The result will be empty if the role does not use
// shared persistent storage.
func getInitContainer(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	pvcNamePrefix string,
	imageID string,
	persistDirs []string,
) (initContainer []v1.Container) {

	// We are depending on the default value of 0 here. Not setting it
	// explicitly because golint doesn't like that.
	var rootUID int64

	if role.Storage == nil {
		return
	}

	initVolumeMounts := generateInitVolumeMounts(pvcNamePrefix)
	cpus, _ := resource.ParseQuantity("1")
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
			SecurityContext: &v1.SecurityContext{
				RunAsUser: &rootUID,
			},
			VolumeMounts: initVolumeMounts,
		},
	}
	return
}

// getVolumeClaimTemplate prepares the PVC templates to be used with the
// given role (for acquiring shared persistent storage). The result will be
// empty if the role does not use shared persistent storage. If the spec contains
// Storage field, a volume Volume Claim with Filesystem volume mode is created. If spec contains a BlockStorage field,
// BlockStorage field, a block Claim with Block volume mode is created.
func getVolumeClaimTemplate(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	pvcNamePrefix string,
) (volTemplate []v1.PersistentVolumeClaim) {

	if role.Storage != nil {
		volSize, _ := resource.ParseQuantity(role.Storage.Size)
		volClaim := v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcNamePrefix,
				Annotations: map[string]string{
					storageClassName: *role.Storage.StorageClass,
				},
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
		}
		volTemplate = append(volTemplate, volClaim)
	}

	if role.BlockStorage != nil {

		block := v1.PersistentVolumeBlock

		blockVolSize, _ := resource.ParseQuantity(defaultBlockDeviceSize)

		if role.BlockStorage.Size != nil {
			blockVolSize, _ = resource.ParseQuantity(*role.BlockStorage.Size)
		}

		numDevices := *role.BlockStorage.NumDevices

		for i := int32(0); i < numDevices; i++ {

			deviceID := strconv.FormatInt(int64(i), 10)
			deviceName := blockPvcNamePrefix + deviceID

			blockClaim := v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: deviceName,
					Annotations: map[string]string{
						storageClassName: *role.BlockStorage.StorageClass,
					},
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceStorage: blockVolSize,
						},
					},
					StorageClassName: role.BlockStorage.StorageClass,

					VolumeMode: &block,
				},
			}

			volTemplate = append(volTemplate, blockClaim)
		}
	}
	return volTemplate
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
				"exec 2>>/tmp/kd-postcluster.log; set -x;" +
					"Retries=60; while [[ $Retries && ! -s /etc/resolv.conf ]]; do " +
					"sleep 1; Retries=$(expr $Retries - 1); done; " +
					"sed \"s/^search \\([^ ]\\+\\)/search " +
					cr.Status.ClusterService +
					".\\1 \\1/\" /etc/resolv.conf > /tmp/resolv.conf.new && " +
					"cat /tmp/resolv.conf.new > /etc/resolv.conf;" +
					"rm -f /tmp/resolv.conf.new;" +
					"chmod 755 /run;" +
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
	// don't do this copy if the kubedirector.init file already exists in /etc.
	launchCmd := "! [ -f /mnt" + kubedirectorInit + " ]" + " && " +
		"cp --parent -ax " + strings.Join(persistDirs, " ") +
		" /mnt; touch /mnt" + kubedirectorInit

	return launchCmd
}

// generateSecretVolume generates VolumeMount and Volume
// object for mounting a secret into a container
func generateSecretVolume(
	secret *kdv1.KDSecret,
) ([]v1.VolumeMount, []v1.Volume) {

	if secret != nil {
		secretVolName := "secret-vol-" + secret.Name
		secretVolumeSource := v1.SecretVolumeSource{
			SecretName:  secret.Name,
			DefaultMode: secret.DefaultMode,
		}
		return []v1.VolumeMount{
				v1.VolumeMount{
					Name:      secretVolName,
					MountPath: secret.MountPath,
					ReadOnly:  secret.ReadOnly,
				},
			}, []v1.Volume{
				v1.Volume{
					Name: secretVolName,
					VolumeSource: v1.VolumeSource{
						Secret: &secretVolumeSource,
					},
				},
			}
	}
	return []v1.VolumeMount{}, []v1.Volume{}

}

// generateVolumeMounts generates all of an app container's volume and mount
// specs for persistent storage, tmpfs and systemctl support that are
// appropriate for members of the given role. For systemctl support,
// nativeSystemdSupport flag is examined along with the app requirement.
func generateVolumeMounts(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	pvcNamePrefix string,
	nativeSystemdSupport bool,
	persistDirs []string,
) ([]v1.VolumeMount, []v1.Volume, error) {
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

	if role.Storage != nil {
		volumeMounts = generateClaimMounts(pvcNamePrefix, persistDirs)
	}

	tmpfsVolMnts, tmpfsVols := generateTmpfsSupport(cr)
	volumeMounts = append(volumeMounts, tmpfsVolMnts...)
	volumes = append(volumes, tmpfsVols...)

	// Generate secret volumes (if needed)
	secretVolMnts, secretVols := generateSecretVolume(role.Secret)
	volumeMounts = append(volumeMounts, secretVolMnts...)
	volumes = append(volumes, secretVols...)

	isSystemdReqd, err := catalog.SystemdRequired(cr)

	if err != nil {
		return volumeMounts, volumes, err
	}

	if isSystemdReqd && !nativeSystemdSupport {
		cgroupVolMnts, cgroupVols := generateSystemdSupport(cr)
		volumeMounts = append(volumeMounts, cgroupVolMnts...)
		volumes = append(volumes, cgroupVols...)
	}

	return volumeMounts, volumes, nil
}

// generateClaimMounts creates the mount specs for all directories that are
// to be mounted from a persistent volume by an app container.
func generateClaimMounts(
	pvcNamePrefix string,
	persistDirs []string,
) []v1.VolumeMount {

	var volumeMounts []v1.VolumeMount
	for _, folder := range persistDirs {
		volumeMount := v1.VolumeMount{
			MountPath: folder,
			Name:      pvcNamePrefix,
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
	pvcNamePrefix string,
) []v1.VolumeMount {

	return []v1.VolumeMount{
		v1.VolumeMount{
			MountPath: "/mnt",
			Name:      pvcNamePrefix,
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

	cgroupFsName := "cgroupfs"
	systemdFsName := "systemd"
	volumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:      cgroupFsName,
			MountPath: cgroupFSVolume,
			ReadOnly:  true,
		},
		v1.VolumeMount{
			Name:      systemdFsName,
			MountPath: systemdFSVolume,
		},
	}
	volumes := []v1.Volume{
		v1.Volume{
			Name: cgroupFsName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: cgroupFSVolume,
				},
			},
		},
		v1.Volume{
			Name: systemdFsName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: systemdFSVolume,
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
			Name:      "tmpfs-tmp",
			MountPath: "/tmp",
		},
		v1.VolumeMount{
			Name:      "tmpfs-run",
			MountPath: "/run",
		},
		v1.VolumeMount{
			Name:      "tmpfs-run-lock",
			MountPath: "/run/lock",
		},
	}
	maxTmpSize, _ := resource.ParseQuantity(tmpFSVolSize)
	volumes := []v1.Volume{
		v1.Volume{
			Name: "tmpfs-tmp",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: &maxTmpSize,
				},
			},
		},
		v1.Volume{
			Name: "tmpfs-run",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium:    "Memory",
					SizeLimit: &maxTmpSize,
				},
			},
		},
		v1.Volume{
			Name: "tmpfs-run-lock",
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

// hasSTDIN is a utility function to find out
// if STDIN was requested by the KubeDirectorApp
// default is False if left blank by the App
func hasSTDIN(
	cr *kdv1.KubeDirectorCluster,
	role string,
) bool {

	containerSpec, _ := catalog.RoleContainerSpecs(cr, role)
	if containerSpec == nil {
		return false
	}

	return containerSpec.Stdin
}

// hasTTY is a utility function to find out
// if TTY was requested by the KubeDirectorApp
// default is False if left blank by the App
func hasTTY(
	cr *kdv1.KubeDirectorCluster,
	role string,
) bool {

	containerSpec, _ := catalog.RoleContainerSpecs(cr, role)
	if containerSpec == nil {
		return false
	}

	return containerSpec.Tty
}
