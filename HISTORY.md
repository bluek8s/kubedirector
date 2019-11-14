# v0.3.0 - ???

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
