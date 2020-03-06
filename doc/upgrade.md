#### From before KubeDirector v0.4.0

We do not have a recommended upgrade process for KubeDirector deployments from before v0.4.0, since they are using alpha versions of the CRDs. The migration approach in this case is a complete teardown and clean re-deploy.

#### From KubeDirector v0.4.0

Follow these steps, in order:

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

One way to make the desired changes in a single update of the resource is to apply a list of JSON patch operations in a single kubectl patch command. For example if you want to use the bluek8s/kubedirector:unstable image, you could use the following command: 
```
kubectl patch deployment kubedirector --type=json -p='[
    {"op": "remove", "path": "/spec/template/spec/containers/0/ports/0"},
    {"op": "remove", "path": "/spec/template/spec/containers/0/env/1/valueFrom"},
    {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "bluek8s/kubedirector:unstable"}
]'
```
Note that this form of patching does make assumptions about which elements of the deployment spec are in which positions in various lists, so it is constructed assuming the exact form of the deployment spec included in this repo. If your deployment spec is different, your required patch specifications may be different.
