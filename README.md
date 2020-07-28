# KubeDirector

[![Build Status](https://travis-ci.com/bluek8s/kubedirector.svg?branch=master)](https://travis-ci.com/bluek8s/kubedirector)

The [**BlueK8s**](https://github.com/bluek8s) open source initiative will include a number of projects to help bring enterprise-level capabilities for distributed stateful applications to Kubernetes. 

The first open source project in this initiative is **Kubernetes Director** or **KubeDirector** for short.

## What is KubeDirector?

**KubeDirector** uses standard Kubernetes (K8s) facilities of custom resources and API extensions to implement stateful scaleout application clusters. This approach enables transparent integration with K8s user/resource management and existing K8s clients and tools.

In broad terms, KubeDirector is a "custom controller" (itself deployed into K8s) that watches for custom resources of a given type to be created or modified within some K8s namespace(s). On such an event, KubeDirector uses K8s APIs to create or update the resources and configuration of a cluster to bring it into accordance with the spec defined in that custom resource.

Unlike some other custom controller implementations, KubeDirector does not tie a custom resource definition to a particular type of application, or contain hardcoded application-specific logic within the controller. Instead, application characteristics are defined by metadata and an associated package of configuration artifacts. This separation of responsibilities has several useful characteristics, including:

* Application experts -- within or outside the organization running KubeDirector -- can enable application deployment without writing "Go" code or understanding the operation of custom controllers. This includes easily making incremental changes to adopt new versions of an application or tweak the setup choices exposed to the end user.

* Site administrators can easily manage which application types and versions are available within an organization, without undergoing a custom controller code upgrade that could potentially disrupt operations.

* End users can launch and reconfigure clusters using familiar K8s tools, selecting from application-specific choices provided to them by the experts.

The [wiki](https://github.com/bluek8s/kubedirector/wiki) describes KubeDirector concepts, architecture, and data formats.

See the files in the "doc" directory for information about deploying and using KubeDirector:
* [quickstart.md](doc/quickstart.md): deploying a pre-built KubeDirector image
* [gke-notes.md](doc/gke-notes.md): important information if you intend to deploy KubeDirector using Google Kubernetes Engine
* [eks-notes.md](doc/eks-notes.md): important information if you intend to deploy KubeDirector using Amazon Elastic Container Service for Kubernetes
* [virtual-clusters.md](doc/virtual-clusters.md): creating and managing virtual clusters with KubeDirector
* [app-authoring.md](doc/app-authoring.md): creating app definitions for KubeDirector virtual clusters
* [kubedirector-development.md](doc/kubedirector-development.md): building KubeDirector from source

# Contributing

You’re welcome to join the [BlueK8s Slack workspace](http://bit.ly/KubeDirectorSlack) for feedback and discussion.

Please read through the [CONTRIBUTING](CONTRIBUTING.md) guide before making a pull request. If you run into an issue with the contributing guide, please send a pull request to fix the contributing guide.

# Code of conduct

Participation in the KubeDirector community is governed by the [KubeDirector Code of Conduct](CODE_OF_CONDUCT.md).
