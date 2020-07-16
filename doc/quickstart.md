#### KUBERNETES SETUP

You will need a K8s (Kubernetes) cluster for deploying KubeDirector and KubeDirector-managed virtual clusters. Currently we require using K8s version 1.14 or later. Especially if you are using a cloud service to spin up K8s clusters, take care that you are getting the necessary K8s version.

We often run KubeDirector on [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine); see [gke-notes.md](gke-notes.md) for GKE-specific elaborations on the various steps in this document. Or if you would rather use [Amazon Elastic Kubernetes Service](https://aws.amazon.com/eks/), see [eks-notes.md](eks-notes.md). We have also run it on [DigitalOcean Kubernetes](https://www.digitalocean.com/products/kubernetes/) without issues.

Cloud K8s services are not the only option. KubeDirector is also deployed by default on every K8s cluster managed by the [HPE Container Platform](https://www.hpe.com/us/en/solutions/container-platform.html); those clusters are currently installed through [kubeadm](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/) using HyperKube images, so any similar K8s installation (or more broadly any installation using open-source K8s images) is likely to be compatible with KubeDirector. The only K8s configuration issue that has historically been a "gotcha" for KubeDirector is that you must ensure that [admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites) are enabled. If you are using kubeadm, you shouldn't have to explicitly worry about that requirement -- its default configuration should be good.

We have typically not recommended KubeDirector deployment on [OpenShift](https://www.openshift.com/) for new KubeDirector users/developers, because of a variety of issues, but this is a topic that we need to revisit since both KubeDirector and OpenShift have undergone relevant changes that should make them a better match.

#### KUBECTL SETUP

You should have kubectl installed on your local workstation, with administrative privileges for deploying resources into some namespace in your K8s cluster (and specifically, setting RBACs there).

Some K8s platforms also provide other ways to run kubectl or manage K8s, but the standard KubeDirector deployment process uses a locally installed kubectl, and the examples in these docs are in terms of using kubectl locally as well. So a local kubectl is necessary. This document does also assume that you have a general familiarity with using common kubectl commands.

We strongly recommend using a kubectl version at least as recent as that of your K8s cluster. You can use "kubectl version" to check this.

#### DEPLOYING KUBEDIRECTOR

Once you have set up a K8s cluster and kubectl, you are ready to deploy a pre-built version of KubeDirector. If you would rather build KubeDirector from source, you will want to read [kubedirector-development.md](kubedirector-development.md) after this doc.

To deploy a pre-built KubeDirector into your K8s cluster, the first step is to clone this repo.

If you think you will eventually be interested in building KubeDirector from source, you need to have ["go"](https://golang.org/) installed and this repo needs to be placed appropriately under your GOPATH. If not however, then you can place this repo anywhere.

So if you intend to later work with the KubeDirector source, you would clone the repo as follows:
```bash
    mkdir -p $GOPATH/src/github.com/bluek8s
    cd $GOPATH/src/github.com/bluek8s
    git clone https://github.com/bluek8s/kubedirector
```
**But** if you intend to only do pre-built deployments, this is fine:
```bash
    cd any_directory_you_like
    git clone https://github.com/bluek8s/kubedirector
```

If you want to work with a specific released version of KubeDirector (instead of the tip of the master branch), now is the time to switch the repo to that. This is recommended, especially for your first time trying out KubeDirector. At the time of last updating this doc, the most recent KubeDirector release was v0.5.0; you can set the repo to that release as follows:
```bash
    cd kubedirector
    git checkout v0.5.0
```

If you have switched to a tagged version of KubeDirector in your local repo, make sure that when you read the doc files (like this one) you reference the files that are consistent with that version. The files in your local repo will be consistent; you could also reference the online files at a particular tag, for example the [doc files for v0.5.0](https://github.com/bluek8s/kubedirector/tree/v0.5.0/doc).

Now you can deploy KubeDirector:
```bash
    make deploy
```

This will create, in the current namespace for your kubectl configuration:
* an administratively-privileged service account used by KubeDirector
* the custom resource definition for KubeDirector virtual clusters
* the custom resource definition for KubeDirector app types
* the custom resource definition for the KubeDirector configuration object
* the KubeDirector deployment itself
* an example set of KubeDirector app types

If you have set the repo to a commit tagged with a KubeDirector release version, then the pre-built KubeDirector deployed in this way will use an image tied to that exact commit. Otherwise the pre-built KubeDirector image will be an "unstable" version associated with the tip of the master branch. If using an "unstable" image you should keep your local copy of the repo close to the tip of master to prevent inconsistencies.

To see the YAML files that are used by "make deploy" to create these resources, look under the "deploy" directory. A few notes:
* The core resources needed to run KubeDirector are in "deploy/kubedirector".
* "deploy/kubedirector/deployment-prebuilt.yaml" will be used to create the KubeDirector deployment if you have **not** built KubeDirector locally. Examine that file and particularly take note of comments about an optional liveness probe.
* "deploy/kubedirector/deployment-localbuilt.yaml" will be used to create the KubeDirector deployment if you **have** built KubeDirector locally. This file is generated at "make build" time from the template at "deploy/operator.yaml", which you may wish to modify if you are building KubeDirector.
* "deploy/kubedirector/rbac.yaml" is generated at "make deploy" time, modifying the template from "deploy/kubedirector/rbac-default.yaml" to use the namespace of your current kubectl context.
* "deploy/example_catalog" contains the example set of KubeDirectorApps.

Once KubeDirector is deployed, you may wish to observe its activity by using "kubectl logs -f" with the KubeDirector pod name (which is printed for you at the end of "make deploy"). This will continuously tail the KubeDirector log.

#### CONFIGURING KUBEDIRECTOR

Before creating any virtual clusters, you may wish to configure KubeDirector to change some default settings. If so, then you can create (in the same K8s namespace as KubeDirector itself) a [KubeDirectorConfig object](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorConfig-Definition) that has the name "kd-global-config".

When using KubeDirector in a standard deployment of Google Kubernetes Engine, DigitalOcean Kubernetes, or Amazon Elastic Container Service for Kubernetes, then no change to the KubeDirector configuration should be necessary. You can still take a look at the [KubeDirectorConfig definition](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorConfig-Definition) to see what configuration properties are available.

If the default KubeDirectorConfig property values look correct for your purposes, then you do *not* need to create a config object.

However if you are using KubeDirector on a local K8s installation that you have installed on top of RHEL/CentOS, you may need to change the values for the defaultServiceType and/or nativeSystemdSupport config properties. See the "deploy/example_configs/cr-config-onprem.yaml" file and particularly the comments at the top of that file. If you want to apply these config values to your deployment, you can use kubectl to create that config resource:
```bash
    kubectl create -f deploy/example_configs/cr-config-onprem.yaml
```

Another common reason you may wish to change the KubeDirector configuration is if you want your clusters to use a persistent storage class that is not the K8s default storage class. You can do this by specifying a value for the defaultStorageClassName property in the config resource.

If you have created a KubeDirectorConfig object and later want to change it, you can edit the config file and use "kubectl apply" to apply the changes. Keep in mind that the values specified in this config are only referenced at the time a virtual cluster is created; changing this config will not retroactively affect any existing virtual clusters.

#### WORKING WITH KUBEDIRECTOR

The process of creating and managing virtual clusters is described in [virtual-clusters.md](virtual-clusters.md).

#### UPGRADING KUBEDIRECTOR

If you have deployed one version of KubeDirector and want to upgrade to a new version, reference [upgrade.md](upgrade.md).

#### TEARDOWN

When you are completely done with KubeDirector, or want to start over fresh, you can delete all KubeDirector-related resources from K8s:
```bash
    make teardown
```

This will delete not only KubeDirector itself but also any KubeDirector-managed virtual clusters and app types that you have created.
