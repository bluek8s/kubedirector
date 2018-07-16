# KubeDirector

The [**BlueK8s**](https://github.com/bluek8s) open source initiative will include a number of projects to help bring enterprise-level capabilities for distributed stateful applications to Kubernetes. 

The first open source project in this initiative is **Kubernetes Director** or **KubeDirector** for short.

## What is KubeDirector?

**KubeDirector** uses standard Kubernetes (K8s) facilities of custom resources and API extension to implement stateful scaleout application clusters. This approach enables transparent integration with K8s user/resource management and existing K8s clients and tools.

In broad terms, KubeDirector is a "custom controller" (itself deployed into K8s) that watches for custom resources of a given type to be created or modified within some K8s namespace(s). On such an event, KubeDirector uses K8s APIs to create or update the resources and configuration of a cluster to bring it into accordance with the spec defined in that custom resource.

Unlike some other custom controller implementations, KubeDirector does not tie a custom resource definition to a particular type of application, or contain hardcoded application-specific logic within the controller. Instead application characteristics are defined by metadata and an associated package of configuration artifacts. This separation of responsibilities has several useful characteristics, including:

* Application experts -- within or outside the organization running KubeDirector -- can enable application deployment without writing "go" code or understanding the operation of custom controllers. This includes easily making incremental changes to adopt new versions of an application or tweak the setup choices exposed to the end user.

* Site administrators can easily manage which application types and versions are available within an organization, without undergoing a custom controller code upgrade that could potentially disrupt operations.

* End users can launch and reconfigure clusters using familiar K8s tools, selecting from application-specific choices provided to them by the experts.

Read the wiki for additional details: https://github.com/bluek8s/kubedirector/wiki

# Roadmap

The first pre-alpha version of KubeDirector is in development now, and will be available soon.

# Contributing

Thanks for your interesting in joining and contributing to our community!

We will update the community on availability of the first pre-alpha release later this summer. 

In the meantime, you’re welcome to join the [BlueK8s Slack workspace](https://bluek8s.slack.com) for feedback and discussion.

# Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](https://github.com/kubernetes/community/blob/master/code-of-conduct.md).
