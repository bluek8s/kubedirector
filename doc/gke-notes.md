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

From here you can proceed to deploy KubeDirector as described in [quickstart.md](quickstart.md). Assuming that you are currently working from the home directory of the KubeDirector repo, the easiest approach is to deploy a prebuilt KubeDirector image, which can be done simply as:
```bash
    make deploy
```

Note that after deploying KubeDirector but before creating virtual clusters, you will want to apply a KubeDirector configuration suitable for GKE:
```bash
    kubectl create -f deploy/example_config/cr-config-gke.yaml
```

Now you can deploy virtual clusters as described in [virtual-clusters.md](virtual-clusters.md).

When you're finished working with KubeDirector, you can tear down your KubeDirector deployment:
```bash
    make teardown
```

If you now want to completely delete your GKE cluster, you can. But be sure to do the KubeDirector teardown as described above before deleting the GKE cluster! Otherwise, some dangling resources may remain in your gcloud project, especially those related to implementing LoadBalancer services. If you are in a development situation where "make teardown" doesn't work for some reason, then use individual "kubectl delete" invocations to delete as many resources as possible (reference how the Makefile implements the teardown action).

Once you have successfully completed the "make teardown" or done the individual resource deletions, you can delete your GKE cluster:
```bash
    gcloud container clusters delete my-gke
```

When you delete the GKE cluster, this will also delete the related context from your kubectl config. If you have some other context that you wish to return to using at this point, you will want to run "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config use-context" to select one.
