#### LIVE APPLICATION UPGRADE

There may appear the situation when the KubeDirector user needs to upgrade the KD application without the stopping all the KD clusters on which this application is performed. As example, to change the used docker image to the newer version. For clarification, this guide describes how to upgrade the performed KD application at the concrete KD cluster without affecting the other KD clusters.

#### 1) Preparing the KD App CR (`<KdAppName>.json` file)

**1.1) `spec.upgradable` field**

Usually there is not possible to change the KD application .json file if this application is performed at least at the one KD cluster. But one of exceptions to this rule is the new boolean field `spec.upgradable` which describes, could the KD cluster with this application instance be upgraded  from the current app version to the newer one.
As `spec.upgradable` field was not previosly present at the existent KD applications CR the KubeDirector versions that already support the live-app-upgrade feature recognize its value as `None`. The developer/maintainer may add it to the application CR and set it to `true` or `false` value. Then the developer/maintainer should apply his changes using 
```bash
    kubectl apply -f <KdOldAppName>.json
```

If `spec.upgradable` value previously was `None` (absent), the KubeDirector validator accepts these changes even when the current application instance is already running at some KD cluster. Otherwise, if `upgradable` field was previously set, it could be changed after all clusters used the current application version were stopped.

**Note**

The `False` value seals the ability to live upgrade clusters which use the current application version.

**1.2) Semantic version checking**

The KD application to be upgraded should have the `spec.version` field value that is compatible with Semantic Versioning rules (see https://semver.org/)

**1.3) New KD application CR requirements**

There are set of strong requirements for the candidate KD application CR. Before the start an upgrade process the KubeDirector validator checks, that the candidate specification has the same `metadata.name` and `spec.distroID`, but the strongly newer `spec.version` values. Also, the KD app developer may change `spec.defaultImageRepoTag` field or `imageRepoTag` field of any role. The developer bears all responsibility of the provided image tags and the application compatibility with the previous version. As the current application may be performed on the several KD clusters, it's strongly recommended to create the copy of the current application CR file and edit the necessary changes including the new application name (a.e. add the version as the suffix to the `metadata.name` field).

After all editings the developer should register the new KD application CR using the command 
```bash
    kubectl apply -f <KdNewAppName>.json
```


#### 2) KD cluster upgrade process

**2.1) Run upgrading**

To start the upgrade process the maintainer should change the `spec.app` field at the KD cluster CR file to the new application name (defined at 1.3)
Then apply the changes using 
```bash
    kubectl apply -f <KdClusterName>.yaml
```

The KD validator will check that the required application is already registered, is upgradable and its version is newer than a current one. Then upgrade process will start for all roles. All the roles are upgrading as parallel processes, pod-by-pod, from the last pod of the role statefulset to the first one. 

**Note**

The upgrade process couldn't be started until the cluster becomes to `configured` state.

It is not possible to update the application name and some other properties (such as the number of role members) of the cluster CR at the same time.

**2.2) Check upgrade status**

Some details of upgrading process the user may observe at the cluster description called by the command 
```bash
    kubectl describe kdcluster <KdClusterName>
```

During upgrade the next fields should appears in the cluster description:
- `Status.State` - `updating`
- `Role.RoleUpgradeStatus` - `RoleUpgrading`/`RoleRollingBack` (for each changing role)
- `Pod.PodUpgradeStatus` - `PodUpgrading`/`PodRollingBack` (for each currently changing pod)
- `UpgradeInfo.IsRollingBack` - `false`/`true` - shows is upgrade in progress or cancelled
- `UpgradeInfo.PrevApp` - stores the name of the previously performed KD application

**2.3) Cancel upgrade (rollback)**

Sometimes there may appear the situation when application couldn't be upgraded by some reason. For example: the new image tag is incorrect or currently does not exist. Then the currently upgrading pod has `ErrImagePull` or `ImagePullBackoff` status error and cluster falls into `updating` infinitely.
For handle this problem the roolback mechanic is provided.
The upgrading cluster may be rolled back strictly to the last KD application was performed before upgrade started. Similarly to the step 2.1 the maintainer should change the `spec.app` field at the KD cluster CR file to the previous one and apply the cluster CR. This name is specified at `UpgradeInfo.PrevApp` KD cluster status field.

**Note**

The rollback couldn't be performed while `UpgradeInfo.IsRollingBack` field is already `true`.


#### 3) KD application behavior customization**

A customer could have some requirements for changes of the performed application from version to version. For example, the new version of application may read or write data to the different place than the old one or may have the totally different file system structure, encodings etc. Anyway, the updated application must be able to work with the old data format or adapt this data for correct work. By this reason there is provided the easy way for developers to describe the post-upgrade actions. The developer just should add the `pod_upgraded` and `pod_reverted` event handlers to the KD application config package `startscript`. More details about `startscript` are described in `app-filesystem-layout.md` - `CONFIG PACKAGE LOCATION` part.

**3.1) `pod_upgraded`**

KubeDirector calls the `startscript` with this argument on the each pod after its image was changed and its `Pod Upgrade Status` field became from `PodUpgrading` to `PodConfigured` value. If the image is not changed during upgrade it won't be triggered.

**3.2) `pod_reverted`**

KubeDirector calls the `startscript` with this argument on the each pod after its `Pod Upgrade Status` field became from `PodRollingBack` to `PodConfigured` value.
