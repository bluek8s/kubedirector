# v0.10.0 - May 13, 2022

The main focus of this release is to remove the usage of some deprecated K8s APIs. As a result of this change, KubeDirector can now be deployed on K8s version 1.22 and later. Also, KubeDirector is dropping support for deploying on K8s versions before 1.16.

We're also starting to qualify KubeDirector for deployment on OpenShift, so you can start experimenting with that environment as of this release.

See the release notes for v0.10.0-beta.1 for more details. There is only one further change in this final release, as follows:

The new API version for CRDs is by default more strict about whether CRs can accept properties that are not present in the CRD's schema. By default such out-of-schema properties will be ignored and not stored in the CR. Depending on your version of kubectl, kubectl may not even allow you to submit a CR with out-of-schema properties unless you use the "--validate=false" flag. This is generally good for reasons described here: https://kubernetes.io/blog/2019/06/20/crd-structural-schema/

KD-related CRDs will be adopting this new stricter behavior. For VERY old legacy reasons however, some of the example kdapp CRs previously included some out-of-schema properties like "categories". These properties are not required for any of KD's own functionality, but some other clients might still look for such properties in a kdapp. Also if a kdapp CR was created from a copy of one of our example CRs, it might still contain such properties even if they are not actually used, creating issues when applying the CR with kubectl as mentioned above.

So _for now_ the kdapp CRD will include declarations to allow and preserve out-of-schema properties in "spec", "spec.label", and the items of the "spec.services" list. This is purely to avoid causing near-term issues with existing kdapp CRs. If you have a kdapp CR that includes out-of-schema properties, please work to remove them. If those properties contain useful information that some client is consuming, consider moving that information to an annotation or label in the CR metadata instead.

These "relaxations" of strictness for out-of-schema properties will be removed for the 1.0.0 release of KubeDirector.

(Note that there is one other instance of allowed out-of-schema properties in the kdapp CRD, and that is in the minResources declaration for a role. This allowance is needed to support extended resource types, and it will not be removed.)

# v0.10.0-beta.1 - Apr 11, 2022

Note that this is a BETA release. It is being made available so that it can be more easily/widely tested.

The main focus of this release is to remove the usage of some deprecated K8s APIs. As a result of this change, KubeDirector can now be deployed on K8s version 1.22 and later. Also, KubeDirector is dropping support for deploying on K8s versions before 1.16.

We're also starting to qualify KubeDirector for deployment on OpenShift, so you can start experimenting with that environment as of this release.

The final non-beta v0.10.0 release (expected before end-of-month) will therefore move the supported-K8s-version window as described above. It will also contain any changes for the initial experimental support of KubeDirector on OpenShift, but it will not yet make OpenShift a recommended environment for deploying KubeDirector.

The specific changes included in this beta:
* [specify additionalPrinterColumns for kdcluster/kdapp](https://github.com/bluek8s/kubedirector/issues/288)
* [stop using deprecated APIs in apiextensions.k8s.io and admissionregistration.k8s.io](https://github.com/bluek8s/kubedirector/issues/504)
* [cooperate with the OwnerReferencesPermissionEnforcement admission controller](https://github.com/bluek8s/kubedirector/issues/583)

# v0.9.0 - Mar 31, 2022

A collection of kdcluster features and bugfixes here. In order of issue number:
* [allow requesting PV storage for kdcluster role members](https://github.com/bluek8s/kubedirector/issues/411); cf. the 0.9.0 additions to the [role spec](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorCluster-Definition#rolespec)
* [don't put owner ref on MutatingWebhookConfiguration](https://github.com/bluek8s/kubedirector/issues/505) since K8s no longer honors this; the kubedirector-webhook needs to be explicitly remove if uninstalling KD ([which "make teardown" now does](https://github.com/bluek8s/kubedirector/issues/570))
* [make configscript error info available in member status](https://github.com/bluek8s/kubedirector/issues/547); cf. the 0.9.0 additions to the [memberStateDetail](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorCluster-Definition#memberstatedetail) in per-member status and to the [kdapp spec](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorApp-Definition#kubedirectorappspec)
* [fix application of the NVIDIA_VISIBLE_DEVICES environment var](https://github.com/bluek8s/kubedirector/issues/507) to properly set this to "void" when no NVidia GPU resources are requested
* [stop using a deprecated method of setting PVC storage class](https://github.com/bluek8s/kubedirector/issues/556); this also fixes some PV backup issues when using Velero
* [make the pod-unschedulable reason available in member status](https://github.com/bluek8s/kubedirector/issues/562); cf. the 0.9.0 additions to the [memberStateDetail](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorCluster-Definition#memberstatedetail) in per-member status
* [give init containers the same resources as app containers](https://github.com/bluek8s/kubedirector/issues/565) to avoid locking down excess resources
* [mark that the KD webhooks support "dry run"](https://github.com/bluek8s/kubedirector/issues/568)

# v0.8.1 - Feb 14, 2022

Happy Valentine's Day! This dot-release is for showing some love to kdapp developers who want to transition to the "new layout" introduced in the prior release, but who will need more time to transition to the more restrictive permissions on /etc/guestconfig. The [app-filesystem-layout.md](doc/app-filesystem-layout.md) doc has been updated with tips. [PR 557](https://github.com/bluek8s/kubedirector/pull/557) describes the corresponding functional change in KubeDirector.

# v0.8.0 - Feb 2, 2022

The primary focus of this release is to add a new option for kdapp CRs that changes the processes and file layout around "app setup script" installation and use: the ["useNewSetupLayout" boolean in the default and per-role config package objects](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorApp-Definition#configpackage). This option can speed up the deployment of PVs in the kdclusters for those apps; it also makes certain directory contents readable only by the container user, for increased security in apps that support login by other user accounts. See the [app-filesystem-layout.md](doc/app-filesystem-layout.md) doc for more info about using this option in your kdapp definitions. Two of the example kdapps (cassandra and jupyter-notebook) have been updated to use this option.

Another new kdapp feature is the ability to optionally specify a minimum valid persistent storage size for each role: the ["minStorage" attribute](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorApp-Definition#role). This helps to prevent cases where a kdcluster requesting a too-small PV cannot come up successfully.

Other changes in this release:
* remove CentOS 8 example kdapp due to EOL
* [handle container images where "rm" is aliased to "rm -i"](https://github.com/bluek8s/kubedirector/issues/530)
* [handle a race in "make deploy" where kdapp creation is attempted too soon](https://github.com/bluek8s/kubedirector/pull/544)


# v0.7.1 - Nov 24, 2021

This dot-release doesn't contain any new features or bugfixes within the KubeDirector code. It updates the base image of the KubeDirector deployment (from UBI7 to UBI8), raises the minimum allowed version of golang for compilation (from 1.13 to 1.16), and updates several of the module dependencies.

If you build KD yourself, the only action required on your part is to make sure you are now using go 1.16 (or later) to compile. Otherwise this update should be transparent to everyone.

As a result of this update, the Trivy report for vulnerabilities in the base image changes from "Total: 655 (LOW: 366, MEDIUM: 284, HIGH: 3, CRITICAL: 2)" to "Total: 96 (LOW: 44, MEDIUM: 48, HIGH: 1, CRITICAL: 3)". The increase in CRITICAL count there is the same CVE-2019-1010022 for glibc. In UBI7 that CVE gets counted against glibc and glibc-common; in UBI8 it's counted against glibc, glibc-common, and glibc-minimal-langpack.

As for Trivy-reported vulnerabilities against the KD binary, those go from "MEDIUM: 3, HIGH: 4" to "MEDIUM: 2, HIGH: 2". Specifically, this update closes out CVE-2020-29652 and CVE-2020-9283 for golang.org/x/crypto, and CVE-2019-11254 for gopkg.in/yaml.v2.

# v0.7.0 - Oct 14, 2021

The big Q2 release is here! Along with some general documentation-cleaning and a new [roadmap](ROADMAP.md), the following features and fixes are included:
* [encrypted secrets](https://github.com/bluek8s/kubedirector/issues/385) also cf. [doc/secret-keys.md](doc/secret-keys.md)
* [support setting up serviceAccounts for role](https://github.com/bluek8s/kubedirector/issues/475)
* [kdcluster backup/restore enablement](https://github.com/bluek8s/kubedirector/issues/500) also cf. [doc/backup-and-restore.md](doc/backup-and-restore.md)
* [race in change processing can cause deadlock](https://github.com/bluek8s/kubedirector/issues/518)

# v0.6.2 - May 21, 2021

Maintenance release to address the following bugs:
* [stderr and stdstatus is never correctly reported while running startscript](https://github.com/bluek8s/kubedirector/issues/482)

# v0.6.1 - May 14, 2021

Maintenance release to address the following bugs:
* [systemd checks report issue in Ubuntu image](https://github.com/bluek8s/kubedirector/issues/417)
* [failure if fractional CPU resource requested for kdcluster](https://github.com/bluek8s/kubedirector/issues/466)
* [possibility of nil deref in handleReadyMembers](https://github.com/bluek8s/kubedirector/issues/467)
* [manual deletion of kdcluster-backing statefulset causes sset spam](https://github.com/bluek8s/kubedirector/issues/470)

Related to the Ubuntu fix, the container image for the example app cr-app-ubuntu18.04.json has also been updated.


# v0.6.0 - Apr 19, 2021

The major change in this release is the addition of a couple of important features -
* Surfacing role level placement constraints for pod. We are surfacing affinity property from pod spec in KubedirectorCluster Spec. [issue #48](https://github.com/bluek8s/kubedirector/issues/48)
* Ability to set STDIN and TTY as part of KubeDirectorApplication role config. [issue #462](https://github.com/bluek8s/kubedirector/pull/462) 

New application(s) in the example catalog
* cr-app-ngc.yaml
* cr-app-mlflow.json


# v0.5.3 - Jan 25, 2021

Maintenance release, primarily to fix [issue #444](https://github.com/bluek8s/kubedirector/issues/444) "surface block device in cluster Pods".

A few changes to the example kdapps catalog as well:

* cr-app-datafabric610.json added
* cr-app-jupyter-notebook.json updated


# v0.5.2 - Oct 28, 2020

Adding support for connections "reconnect" notifications via startscript ([issue #395](https://github.com/bluek8s/kubedirector/issues/395)) and updating configcli to support Python 3 ([bluek8s/configcli#4](https://github.com/bluek8s/configcli/issues/4)).


# v0.5.1 - Aug 26, 2020

Maintenance release, primarily to fix [issue #387](https://github.com/bluek8s/kubedirector/issues/387) "possible to specify kdcluster name that prevents member pod creation".

A few changes to the example kdapps catalog as well:

* cr-app-gitlab-ee.json added
* cr-app-tensorflow-cpu-dockerhub.json added
* cr-app-mapr610-secured.json updated
* cr-app-mapr610-unsecured.json removed


# v0.5.0 - Jul 16, 2020

The major change in this release is the move from operator SDK v0.8.1 to v0.15.2, which has a couple of notable sets of consequences:
* Anyone building KubeDirector from source, as opposed to using a pre-built operator image, should re-read [doc/kubedirector-development.md](doc/kubedirector-development.md) to become acquainted with new requirements and with some tips for continuing to build your existing local repo. We highly recommend reading this **before** merging the new release source into your existing source.
* Anyone who is currently running a KubeDirector 0.4.x deployment and is interested in updating that deployment in-place should read [doc/upgrade.md](doc/upgrade.md).

KD 0.5.0 supports some additional properties in [KubeDirectorApp](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorApp-Definition) and [KubeDirectorCluster](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorCluster-Definition); see the "properties supported by KD v0.5.0+" sections in those tables. These implement three new features:
* A KD app can indicate that a unique token should be generated to be used for authentication to a service endpoint. This token will be visible in the KD cluster member status, visible within the relevant container (through configcli) for use in setting up that service, and also advertised through an annotation on the K8s Service resource for that member.
* A KD app can define which lifecycle events (among "configure"/"addnodes"/"delnodes") the members in a role actually care about being notified for. If we can skip sending notifications to a member, then it matters less if that member is down when a KD cluster is reconfigured.
* A KD cluster can specify a list of "connections" to certain other resources (secrets, configmaps, and/or other KD clusters) whose configuration will be shared within the container (through confligcli). This info will be updated as the connections list is edited or the resources are modified. See the wiki for more details.

The [example catalog of KD apps](https://github.com/bluek8s/kubedirector/tree/master/deploy/example_catalog) has also been significantly reworked and expanded, including additional complex applications and a collection of [usage documentation](https://github.com/bluek8s/kubedirector/tree/master/deploy/example_catalog/docs). We know we have more to do on the front of app-development documentation and that will be a major focus of the next release!

A final bonus QOL improvement is included for automated deployment of KD app CRs. The KD app validation will now allow a PUT to an existing KD app CR that is being used by existing KD clusters, if it is determined that the resulting document (after defaults-substitutions and any other mutations) will not end up changing the "spec" section that the clusters depend on.


# v0.4.2 - Jun 20, 2020

Capturing some bugfixes here while we continue to work toward the 0.5 release, which will introduce new features, add new example kdapps, and move to a newer version of the operator SDK.

Issues addressed in this release:

* [GPU visible in non-GPU-requesting pod](https://github.com/bluek8s/kubedirector/issues/263)

* [configure.stdout/stderr are empty in container](https://github.com/bluek8s/kubedirector/issues/286)

* [don't update LastSetupGeneration until all members have exited transitional state](https://github.com/bluek8s/kubedirector/issues/308)

* [set SpecGenerationToProcess earlier](https://github.com/bluek8s/kubedirector/issues/312)

* [ensure systemd is up before service enable/start](https://github.com/bluek8s/kubedirector/issues/315)

* [don't include zero-member roles in the generated configmeta](https://github.com/bluek8s/kubedirector/issues/317)


# v0.4.1 - Apr 14, 2020

Fix for https://github.com/bluek8s/kubedirector/issues/289


# v0.4.0 - Feb 24, 2020

This is a major release in a few interesting ways. We've had the chance to get wider use of and feedback on KubeDirector, which has led us to these changes.

First and most obviously, we've moved the API to "beta" rather than "alpha" status, starting as "v1beta1". This means that future releases of KD will not make any changes to the "v1beta1" resource definitions. Also, even when we introduce newer API versions, KD will continue to support "v1beta1" for some time (exact support schedule TBD). If you are familiar with the alpha version of the API, please pay close attention to the new CRDs and their documentation on the wiki.

The namespace of the API has also moved from "kubedirector.bluedata.io" to "kubedirector.hpe.com", to use a domain that we can hold on to going forward.

Another significant change is in the way that KD handles member pods that become unresponsive, either temporarily or permanently. Note that restarting a permanently unresponsive member (likely because of K8s node-down) is still an explicit "delete pod" action by the user; see [issue #274](https://github.com/bluek8s/kubedirector/issues/274) for tracking possible future features in that area. However KD will now automatically take the following actions:
* If a member is unresponsive and its role did not request persistent storage, KD will perform a from-scratch re-run of its setup script if/when it comes back up as a new container.
* If a member in "config error" state is restarted, KD will perform a from-scratch re-run of its setup script when it comes back up as a new container.
* KD will be persistent when attempting to update the "configmeta" JSON inside a container and when attempting to notify a container's setup script of changes. If the member is unresponsive when such an update is attempted, or becomes so during the update, it will be reconfigured and re-notified as necessary if/when it comes back up as a new container.

Finally, a number of properties were added to the status stanza of a KD cluster CR, to advertise additional fine-grained per-member status as well as provide a top-level block of "rollup" status that can be quickly scanned to see what is going on.

The wiki will be updated in the near future to provide more information about these changes and what they mean for app setup, but in the meantime interested parties can refer to the description of [PR #272](https://github.com/bluek8s/kubedirector/pull/272) for some details.

We should also mention an [issue that can arise with KD clusters](https://github.com/coredns/coredns/issues/3693) when using CoreDNS as the DNS service in your K8s cluster. Because this issue has been [resolved on CoreDNS master branch](https://github.com/coredns/coredns/pull/3687) we have chosen not to delay this KD release in search of a workaround.

A complete list of changes in this release:

## App/cluster model

* Additional per-member and rollup status properties as described above, in the wiki, and in [PR #272](https://github.com/bluek8s/kubedirector/pull/272).

* The stable status for the overall KD cluster and each member has been renamed from "ready" to "configured".

* Some changes to the semantics of specifing persistDirs in the KD app resource. This list of directories now explicitly represents the persistence needs of the app; KD may also decide to persist additional directories to support its own operations.

## Operational

* Improved handling for unresponsive and/or rebooted members as described above, in the wiki, and in [PR #272](https://github.com/bluek8s/kubedirector/pull/272).

* Post-container-start modifications to /etc/resolv.conf can now survive certain races in container bringup and mounts.

* Fixed operator-restart problems caused when GET through the split K8s client initially returns 404 for resources that actually exist.

* Fixed behavior of persistent storage in cases where a directory in persistDirs does not exist (was causing storage to be re-initialized on member restart).

* Support for a liveness probe via the /healthz URL on port 8443. See the comments in the deployment YAML for an example of how to enable the probe, and why you might not want to do this during development.

* Regularized the naming of generated objects:
  * headless service is named kdhs-\<hs-unique>
  * statefulset is named kdss-\<ss-unique>
  * pod in statefulset is named kdss-\<ss-unique>-\<podnum>
  * service exposing a pod's ports is named s-kdss-\<ss-unique>-\<podnum>
  * PVC persisting a pod's data is named p-kdss-\<ss-unique>-\<podnum>

* Regularized the labelling of generated objects:
  * Labels on any statefulset, pod, or service (either per-member or headless) created by KD:
    * kubedirector.hpe.com/kdcluster: \<kdcluster resource name>
    * kubedirector.hpe.com/kdapp: \<kdapp resource name>
    * kubedirector.hpe.com/appCatalog: \<"local" or "system">
  * Labels on any statefulset, pod, or per-member service created by KD:
    * kubedirector.hpe.com/role: \<kdapp role ID>
  * Labels on any statefulset or pod created by KD:
    * kubedirector.hpe.com/headless: \<name of headless cluster service>

* Annotation on any statefulset, pod, or service created by KD:
  * kubedirector.hpe.com/kdapp-prettyName: \<KD app label name>


## Developer support

* Added "shortname" variants of the CRs: kdapp, kdcluster, and kdconfig.

* Made updates to the catalog of example kdapps, especially fixes for the example TensorFlow app.

* "make compile" changed to use the trimpath option. Also cf. [issue #266](https://github.com/bluek8s/kubedirector/issues/266).


# v0.3.2 - Jan 30, 2020

As sometimes happens, it looks like we need a fix for one of those fixes. Cf. [issue #253](https://github.com/bluek8s/kubedirector/issues/253).

If you are coming to this release straight from v0.3.0, make sure to also read the change history for v0.3.1 below.


# v0.3.1 - Jan 29, 2020

Primarily we're pushing this release to make some bugfixes public, but there are a few other nice changes included.

Note that our baseline for supported Kubernetes versions has been raised to 1.14. This was necessary to have support for proper CR schema validation. If you're creating your K8s cluster using a cloud service, pay close attention to the version of K8s that you request.

Also note that in the next release we will be changing the API namespace for our CRs from "kubedirector.bluedata.io/v1alpha1" to "kubedirector.hpe.com/v1beta1". We have _not_ yet made this change; this is just a heads-up.

## App/cluster model

* Better schema-based validation for CRs.

* New optional clusterSvcDomainBase property in the KubeDirectorConfig spec.

* New optional podLabels and serviceLabels property in role spec, to place specified labels on generated resources.

## Operational

* Fixed some DNS issues by setting publishNotReadyAddresses=true for the headless cluster service.

* New port naming format for ports on generated service resources; port name is now prefixed with the urlScheme from the app service definition ("generic-" prefix if no urlScheme).

* Liveness probe added for the operator pod.

* Changes to CRs will now be rejected if the operator itself is down.

* Some fixes for proper handling of the persistent storage in KDClusters that don't use a config package, and some fixes for those that do.

## Developer support

* Many updated docs, especially for GKE and EKS.

* Changed example apps to include the config package on the container image, for easier deployment in K8s clusters that don't have S3 access.

* New TensorFlow example app (old one removed).

* Some reduction of log verbosity.


# v0.3.0 - Nov 16, 2019

The most extensive change in this release is that the version of the operator SDK used by KubeDirector has been updated from v0.0.6 to v0.8.1. This picks up the operator SDK's transition to being based on the Kubernetes controller-runtime project. The effects from that change propagated into most corners of the KubeDirector codebase; while the functional difference is not generally visible to the end user, this should put us on a better footing for future updates and maintenance.

In another case of "taking our medicine", the property names in our CRs have all been regularized to use camelCase style. This means that any existing CR YAML or JSON will need to be updated accordingly. The example CRs in this repo, in the various subdirectories of "deploy", have of course been updated. As usual, note that the [wiki](https://github.com/bluek8s/kubedirector/wiki) documents the release-specific format of each CRD; during the "alpha" phase of this API they may still change freely in non-backward-compatible ways between releases.

Also note that the list of "always mounted" directories when requesting persistent storage has been reduced; now only the "/etc" directory is a required mount, and anything else must be explicitly listed in the KubeDirectorApp CR. This can help drastically reduce startup time in cases where "/usr", "/opt", and/or "/var" do not need to be persistent.

Final general comment: our baseline for supported Kubernetes versions has been raised to 1.12. Also, if you are compiling KubeDirector yourself you must now use version 1.12 or later of the go language. The fact that those version numbers match is purely coincidental!

Now on to the feature bulletpoints:

## App/cluster model

* The status stanza for a KubeDirectorCluster is now a subresource. KubeDirectorConfig has also gained a status subresource.

* KubeDirectorCluster resources can now specify AMD or NVidia GPU consumption.

* A KubeDirectorApp can now specify minimum resource requirements per role, which will be enforced when validating a KubeDirectorCluster.

* A KubeDirectorCluster can be deployed into any namespace (not limited to the namespace where KubeDirector resides).

* A KubeDirectorCluster can reference a KubeDirectorApp that exists either in its own namespace or in the KubeDirector namespace.

* Each role in a KubeDirectorCluster may specify a URL for a file to be downloaded into each of that role's containers.

* Each role in a KubeDirectorCluster may specify a K8s secret to be mounted into each of that role's containers. Note that doing so necessarily reveals the secret's name to anyone who has GET privileges for various K8s resource types; see the wiki for more discussion of this feature.

## Operational

* The KubeDirector container image is now based on ubi-minimal (rather than on Alpine).

* KubeDirector now runs as a non-root user.

## Developer support

* The ConfigCLI materials have been migrated to [their own git repository](https://github.com/bluek8s/configcli).

* More documentation and Makefile support for EKS deployments. Cf. [doc/eks-notes.md](doc/eks-notes.md).

* The result of "make compile" can now be used for "make redeploy" even if done on non-Linux.

* Various improvements to Makefile deploy/teardown targets to make them more bulletproof and less verbose.

* The generated deepcopy code is no longer committed to git, so it won't cause merge conflicts.


# v0.2.0 - Feb 19, 2019

Welcome to the first public update of KubeDirector! This release focuses on making the app and virtual cluster CRs more complete, consistent, and bulletproof to use. In the process other operational improvements have fallen into place, and of course bugfixing is always going on.

We still support Kubernetes version 1.9 or later, but our testing has focussed on 1.11 or later. It is likely that a near-future KubeDirector release will raise the minimum supported Kubernetes version to 1.11.

If you have used KubeDirector previously, take note of one change in particular: some virtual cluster properties with usually-constant values can now have defaults defined in a global KubeDirector config. These properties have been removed from the example virtual cluster CRs so that the same example CRs can be used across multiple platforms. This is a nice reduction of complexity, but it does mean that if you don't have the correct global config for your KubeDirector deployment, then virtual clusters created from the example CRs may not work. Fortunately the config defaults should be fine for most platforms. See the new "CONFIGURING KUBEDIRECTOR" section in the [quickstart doc](doc/quickstart.md) for details.

That's far from the only user-visible change though! The entire list of improvements can be bucketed into the following three categories. Note that any use of the term "cluster" below refers to the virtual clusters created/managed through KubeDirector.

## App/cluster model

* KubeDirector-related custom resource definitions (CRDs) have changed in various ways since the v0.1.0 release. The [wiki](https://github.com/bluek8s/kubedirector/wiki) documents the release-specific format of each CRD (as well as the format in use on the tip of the master branch). Use that documentation as a reference if you need to port old CR specs to v0.2.0. We can also provide porting assistance on our [Slack workspace](https://join.slack.com/t/bluek8s/shared_invite/enQtNTQzNDQzNjQwMDMyLTdjYjE0ZTg0OGJhZWUxMzhkZTZjNDg5ODIyNzZmNzZiYTk4ZjQxNDFjYzk4OWM0MjFlNmVkNWNlNmFjNzkzNjQ).

* The CRD schemas and the dynamic validation perform even more validation on app and cluster CRs. It should be extremely unlikely now that an invalid app/cluster CR creation or edit will pass validation.

* A global KubeDirector configuration is now supported. You can use this to specify defaults for some common cluster properties like service type and persistent storage class. This config object is documented on the [wiki](https://github.com/bluek8s/kubedirector/wiki) and in the [quickstart doc](doc/quickstart.md).

* If neither the global configuration nor the cluster spec identify a storage class to use for persistent storage, KubeDirector will now fall back to using the storage class identified as default by the underlying k8s platform.

* The app CR now supports shipping the "app setup package" on the container image itself, or even not using app setup at all.

* The persist_dirs list in an app CR (for persistent storage mounts) can now be specified at a per-role granularity if you like.

* You can now edit the serviceType of an existing cluster CR, and the type of its existing service resources will be changed accordingly.

* Ports on service resources now have more useful names, taken from the service IDs specified in the app CR.

## Operational

* The cluster member launch and configuration processes have been further parallelized. Things get done faster!

* If a cluster edit has requested X additional members but only some smaller number Y have come up so far, at the next polling interval KubeDirector will go ahead and handle those Y members as a complete resize operation in and of themselves. (It will of course keep working toward fulfilling the entire requested member count.) Essentially, big expansions may be implemented in phases to allow resources to come "online" in the cluster as soon as they are available.

* Various happenings during a cluster create or modify operation are now recorded as k8s events on the cluster CR. You can see these via "kubectl describe" for that CR.

* K8s clients are now prohibited from deleting or editing an app CR if any current cluster CRs are referencing that app.

## Developer support

* The "deploy/example_catalog" directory has gained a new very simple example app (vanilla CentOS 7) and a new complicated one (CDH 5.14.2 with Pig, Hive, and Hue support).

* The "configcli" tool used in app setup is now included in this repo, in the "nodeprep" directory.

* Makefile improvements: KubeDirector can now be built and deployed from Ubuntu systems, "make deploy" now waits for deployment to succeed before returning, and "make teardown" now waits for teardown to finish before returning.

* We have tested using KubeDirector on DigitalOcean Kubernetes (DOK). We haven't done extensive work on DOK yet, but we haven't found any issues that would prevent running KubeDirector and its managed clusters there in a manner similar to doing so on GKE. We do recommend using K8s version 1.11 or later on this service.

* We have also tested using KubeDirector on Amazon Elastic Container Service for Kubernetes (EKS). However we have identified [an issue with using persistent storage on EKS](https://github.com/bluek8s/kubedirector/issues/132), which we may be able to address once EKS supports K8s version 1.12. In the meantime, clusters that do *not* use persistent storage should work there.

* All sorts of improvements and additions to the docs in this repo as well as on the wiki!


# v0.1.0 - Oct 2, 2018

Initial public release.
