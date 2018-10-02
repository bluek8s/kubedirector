If you intend to deploy KubeDirector on GKE, you will need to have a Google Cloud Platform project set up with Compute Engine and Kubernetes Engine APIs enabled. You must also have kubectl ready to use.

If you're starting from scratch with GKE, the first few sections of the [GKE Quickstart](https://cloud.google.com/kubernetes-engine/docs/quickstart) may be useful, but you should probably stop after the "Configuring default settings for gcloud" section in that page and return here.

With gcloud configured to use the appropriate project, you can then launch a GKE cluster. For example, this gcloud command will create a 3-node GKE cluster named "my-gke":
```bash
gcloud container clusters create my-gke --machine-type n1-highmem-4
```
(See [the Machine Types list](https://cloud.google.com/compute/docs/machine-types) for the details of the available GKE node resources.)

If you need to grow your GKE cluster you can use gcloud to do that as well; for example, growing to 5 nodes:
```bash
gcloud container clusters resize my-gke --size=5
```

Once your GKE cluster has been created, you will need to set up your kubectl credentials to access it. First, create a kubectl config context for the cluster:
```bash
gcloud container clusters get-credentials my-gke
```

And to deploy KubeDirector into this cluster, you will need for the user in that kubectl context (which is tied to your Google account credentials) to have the cluster-admin role in the cluster.
```bash
# This should be the email that is associated with the Google account that
# gcloud is using.
ACCOUNT="foo@bar.com"
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=${ACCOUNT}
```

From here you can proceed to deploy KubeDirector and work with virtual clusters normally. Cf. the other doc files such as [quickstart.md](quickstart.md) and [virtual-clusters.md](virtual-clusters.md).

When you're finished, you can destroy the GKE cluster:
```bash
gcloud container clusters delete my-gke
```

This will also delete the related context from the kubectl config.

If you have some other context that you wish to return to using at this point, you will want to run "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config use-context" to select one.
