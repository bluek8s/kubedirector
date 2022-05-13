#### First, check Kubernetes version requirements

From KubeDirector v0.4.0 up through v0.8.1, the minimum supported K8s version was 1.14 and the maximum was 1.21.

Starting with KD v0.10.0, the minimum supported K8s version is 1.16 and there is no maximum supported K8s version (yet).

Select your desired KubeDirector version accordingly, and possibly also coordinate with any desired K8s upgrades.


#### If upgrading from KubeDirector v0.5.0 or later:

**1) Update the Deployment resource named "kubedirector".**

The deployment needs to be updated to use the current container image. There are various ways to do this; "kubectl apply" with the latest deployment YAML is the simplest. E.g. if you are using deployment-prebuilt.yaml, then (assuming a kubectl context that is operating in the correct namespace):
```
kubectl apply -f deployment-prebuilt.yaml
```

**2) Update the CRDs.**

Replace the CRDs for kubedirectorconfig, kubedirectorapp, kubedirectorcluster, and kubedirectorstatusbackup with the current version. E.g., while in the deploy/kubedirector directory:
```
kubectl replace -f kubedirector.hpe.com_kubedirectorconfigs_crd.yaml
kubectl replace -f kubedirector.hpe.com_kubedirectorapps_crd.yaml
kubectl replace -f kubedirector.hpe.com_kubedirectorclusters_crd.yaml
kubectl replace -f kubedirector.hpe.com_kubedirectorstatusbackups_crd.yaml
```

Note that specifically using "kubectl replace" (rather than "kubectl apply") is recommended to get a clean update of the CRD. If you are upgrading from a release before 0.7.0 where the kubedirectorstatusbackups CRD does not yet exist, you can use "kubectl create" for that one.

#### If upgrading from KubeDirector v0.4.x:

**1) Remove the Service resource named "kubedirector".**

This is a metrics-related service that is not relevant to newer versions. Assuming your kubectl context is operating in the namespace where KubeDirector is deployed, the following command will remove this service:
```
    kubectl delete service kubedirector
```

**2) Update the ClusterRole resource named "kubedirector".**

The newer build requires additional privileges.

The easiest approach is to make a temporary copy of the new "rbac-default.yaml" file (in deploy/kubedirector), and edit this copied file to remove the ServiceAccount and ClusterRoleBinding resources at the bottom of the file, leaving only the new ClusterRole resource. Then you can apply this file to edit the existing ClusterRole resource. If for example you called your temporary file "rbac-copy.yaml", you could apply the change with this command:
```
    kubectl apply -f rbac-copy.yaml
```
You can then delete this temporary file.

**3) Update the Deployment resource named "kubedirector".**

The existing KubeDirector deployment can now be updated to the new version. This can be done using "kubectl edit", or more programmatically in various ways. The three changes needed are:
* Remove the metrics container port.
* Set the value of WATCH_NAMESPACE to explicit emptystring.
* Change the container image.

There are various ways to apply these changes, but keep in mind that every edit to a deployment will cause its pod(s) to be restarted, so it is preferable to do this in a small number of changes. Unfortunately just doing a "kubectl apply" of a complete spec such as deploy/kubedirector/deployment-prebuilt.yaml or deploy/kubedirector/deployment-localbuilt.yaml will not properly update the container ports or env.

One way to make the desired changes in a single update of the resource is to apply a list of JSON patch operations in a single kubectl patch command. For example if you want to use the bluek8s/kubedirector:unstable image, you could use the following command (assuming a kubectl context that is operating in the correct namespace):
```
kubectl patch deployment kubedirector --type=json -p='[
    {"op": "remove", "path": "/spec/template/spec/containers/0/ports/0"},
    {"op": "remove", "path": "/spec/template/spec/containers/0/env/1/valueFrom"},
    {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "bluek8s/kubedirector:unstable"}
]'
```
Note that this form of patching does make assumptions about which elements of the deployment spec are in which positions in various lists, so it is constructed assuming the exact form of the deployment spec included in this repo. If your deployment spec is different, your required patch specifications may be different.

**3) Update the CRDs.**

Replace the CRDs for kubedirectorconfig, kubedirectorapp, and kubedirectorcluster with the latest version. E.g., while in the deploy/kubedirector directory:
```
kubectl replace -f kubedirector.hpe.com_kubedirectorconfigs_crd.yaml
kubectl replace -f kubedirector.hpe.com_kubedirectorapps_crd.yaml
kubectl replace -f kubedirector.hpe.com_kubedirectorclusters_crd.yaml
```


#### If upgrading from before KubeDirector v0.4.0:

We do not have a recommended upgrade process for KubeDirector deployments from before v0.4.0, since they are using alpha versions of the CRDs. The migration approach in this case is a complete teardown and clean re-deploy.
