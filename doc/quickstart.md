#### KUBERNETES SETUP

You will need a K8s (Kubernetes) cluster for deploying KubeDirector and KubeDirector-managed virtual clusters. Currently we require using K8s version 1.9 or later. We have run KubeDirector both on GKE (see [gke-notes.md](gke-notes.md)), on K8s installed on our own datacenter hosts using RPMs from kubernetes.io, and experimentally on DigitalOcean Kubernetes. If you are installing K8s yourself instead of using GKE, note that you will need to ensure that [admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites) are enabled.

You should have kubectl installed on your local workstation, with administrative privileges for deploying resources into some namespace in your K8s cluster (and specifically, setting RBACs there). This document does also assume that you have familiarity with using common kubectl commands.

We strongly recommend using a kubectl version at least as recent as that of your K8s cluster. You can use "kubectl version" to check this.

That is the only setup necessary for deploying a pre-built version of KubeDirector, which will be described below. If you would rather build KubeDirector from source, you will want to read [kubedirector-development.md](kubedirector-development.md) after this doc.

#### DEPLOYING KUBEDIRECTOR

You can deploy a pre-built KubeDirector into your K8s cluster. First, you need to clone this repo.

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

If you want to work with a specific released version of KubeDirector (instead of the tip of the master branch), now is the time to switch the repo to that. This is recommended, especially for your first time trying out KubeDirector. At the time of last updating this doc, the most recent KubeDirector release was v0.1.0; you can set the repo to that release as follows:
```bash
    cd kubedirector
    git checkout v0.1.0
```

Now you can deploy KubeDirector:
```bash
    make deploy
```

This will create, in the current namespace for your kubectl configuration:
* an administratively-privileged service account used by KubeDirector
* the custom resource definition for KubeDirector virtual clusters
* the custom resource definition for KubeDirector app types
* the KubeDirector deployment itself
* custom app types

If you have set the repo to a commit tagged with a KubeDirector release version, then the pre-built KubeDirector deployed in this way will use an image tied to that exact commit. Otherwise the pre-built KubeDirector image will be an "unstable" version associated with the tip of the master branch. If using an "unstable" image you should keep your local copy of the repo close to the tip of master to prevent inconsistencies.

Once KubeDirector is deployed, you may wish to observe its activity by using "kubectl logs -f" with the KubeDirector pod name (which is printed for you at the end of "make deploy"). This will continuously tail the KubeDirector log.

KubeDirector is now running. You can create and manage virtual clusters as described in [virtual-clusters.md](virtual-clusters.md). But first you may want to set a default configuration for some cluster properties.

#### CONFIGURING KUBEDIRECTOR

Before creating any virtual clusters, you should configure KubeDirector to set some defaults. This is done by creating a [KubeDirectorConfig object](https://github.com/bluek8s/kubedirector/wiki/App-Definition-Authoring-for-KubeDirector). Example KubeDirectorConfig objects are provided in the "deploy/example_config" directory for Google Kubernetes Engine ("cr-config-gke.yaml"), for a generic local K8s installation ("cr-config.yaml"), for DigitalOcean Kubernetes ("cr-config-dok.yaml"), and for OpenShift ("cr-config-okd.yaml"). (Note however that OpenShift deployments are not currently officially supported; cf. the [known issues](https://github.com/bluek8s/kubedirector/issues/1)). You can use one of these example configs or create one that is tailored to your environment.

For example, typically for a GKE deployment you would execute this command:
```bash
    kubectl create -f deploy/example_config/cr-config-gke.yaml
```

If you want to change this configuration at any time, you can edit the config file and use "kubectl apply" to apply the changes. Keep in mind that the defaults specified in this config are only referenced at the time a virtual cluster is created; changing this config will not retroactively affect any existing virtual clusters.

#### TEARDOWN

When you are completely done with KubeDirector, or want to start over fresh, you can delete all KubeDirector-related resources from K8s:
```bash
    make teardown
```

This will delete not only KubeDirector itself but also any KubeDirector-managed virtual clusters and app types that you have created.
