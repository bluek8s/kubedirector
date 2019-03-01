# v0.3.0 - ???

* ConfigCli is migrated to its own Git repository

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
