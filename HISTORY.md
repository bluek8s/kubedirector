# 0.2.0 - ??? insert release date here when finalized ???

Welcome to the first public update of KubeDirector! This release focuses on making the app and virtual cluster CRs more complete, consistent, and bulletproof to use. In the process other operational improvements have fallen into place, and of course bugfixing is always going on.

If you have used KubeDirector previously, take note of one change in particular: some virtual cluster properties with usually-constant values can now have defaults defined in a global KubeDirector config. These properties have been removed from the example virtual cluster CRs so that the same example CRs can be used across multiple platforms. This is a nice reduction of complexity, but it does mean that if you don't apply the correct global config to your KubeDirector deployment, then virtual clusters created from the example CRs may not work. See new "CONFIGURING KUBEDIRECTOR" section in the [quickstart doc](doc/quickstart.md).

That's far from the only user-visible change though! The entire list of improvements can be bucketed into the following three categories. Note that any use of the term "cluster" below refers to the virtual clusters created/managed through KubeDirector.

## App/cluster model

* A global KubeDirector configuration is now supported. You can use this to specify defaults for some common cluster properties like service type and persistent storage class. This config object is documented on the [wiki](https://github.com/bluek8s/kubedirector/wiki) and in the [quickstart doc](doc/quickstart.md), and examples are provided in the "deploy/example_config" directory.

* The CRD schemas and the dynamic validation perform even more validation on app and cluster CRs. It should be extremely unlikely now that an invalid app/cluster CR creation or edit will pass validation.

* You can now edit the serviceType on existing cluster CR, and the type of its existing service CRs will be changed accordingly.

* The persist_dirs list in an app CR (for persistent storage mounts) can now be specified at a per-role granularity if you like.

* The app CR now supports shipping the "app setup package" on the container image itself, or even not using app setup at all.

* Ports on service CRs now have more useful names, taken from the service IDs specified in the app CR.

## Operational

* The cluster member launch and configuration processes has been further parallelized. Things get done faster!

* If a cluster edit has requested X additional members but only some smaller number Y have come up so far, at the next polling interval KubeDirector will go ahead and handle those Y members as a complete resize operation in and of themselves. (It will of course keep working toward fulfilling the entire requested member count.) Essentially, big expansions may be implemented in phases to allow resources to come "online" in the cluster as soon as they are available.

* Various happenings during a cluster create or modify operation are now recorded as k8s events on the cluster CR. You can see these via "kubectl describe" for that CR.

* K8s clients are now prohibited from deleting or editing an app CR if any current cluster CRs are referencing that app.

## Developer support

* The "deploy/example_catalog" directory has gained a new very simple example app (vanilla CentOS 7) and a new very complicated one (CDH 5.14.2 with Pig, Hive, and Hue support).

* The "configcli" tool used in app setup is now included in this repo, in the "nodeprep" directory.

* Makefile improvements: KubeDirector can now be built and deployed from Ubuntu systems, and "make deploy" now waits for deployment to succeed before returning.



# 0.1.0 - Oct 2, 2018

Initial public release.