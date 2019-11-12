#### KUBERNETES SETUP

If you intend to deploy KubeDirector on GKE, you will need to have a Google Cloud Platform project set up with Compute Engine and Kubernetes Engine APIs enabled. You must also have kubectl ready to use.

If you're starting from scratch with GKE, the first few sections of [Google's GKE Quickstart guide](https://cloud.google.com/kubernetes-engine/docs/quickstart) may be useful, but you should probably stop after the "Configuring default settings for gcloud" section in that page and return here.

With gcloud configured to use the appropriate project, you can then launch a GKE cluster. For example, this gcloud command will create a 3-node GKE cluster named "my-gke":
```bash
    gcloud container clusters create my-gke --machine-type n1-highmem-4
```
(See [the Machine Types list](https://cloud.google.com/compute/docs/machine-types) for the details of the available GKE node resources.)

If you need to grow your GKE cluster you can use gcloud to do that as well; for example, growing to 5 nodes:
```bash
    gcloud container clusters resize my-gke --num-nodes=5
```

#### KUBECTL SETUP

Once your GKE cluster has been created, you will need to set up your kubectl credentials to access it. Assuming you already have kubectl installed locally, you can create a kubectl config context for the cluster:
```bash
    gcloud container clusters get-credentials my-gke
```

To deploy KubeDirector into this cluster, you will also need for the user in that kubectl context (which is tied to your Google account credentials) to have the cluster-admin role in the cluster.
```bash
    # This should be the email that is associated with the Google account that
    # gcloud is using.
    ACCOUNT="foo@bar.com"
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=${ACCOUNT}
```

#### DEPLOYING KUBEDIRECTOR

From here you can proceed to deploy KubeDirector as described in [quickstart.md](quickstart.md).

#### CONFIGURING KUBEDIRECTOR

After deploying KubeDirector but before creating virtual clusters, you may wish to create a KubeDirectorConfig object as described in [quickstart.md](quickstart.md). However, it is likely that the default values for the current config properties will be fine for your GKE deployment, and you can skip this step. (Future releases of KubeDirector may add more config properties that may be more likely to vary across GKE deployments.)

#### WORKING WITH KUBEDIRECTOR

The process of creating and managing virtual clusters is described in [virtual-clusters.md](virtual-clusters.md).

#### TEARDOWN

When you're finished working with KubeDirector, you can tear down your KubeDirector deployment:
```bash
    make teardown
```

If you now want to completely delete your GKE cluster, you can. But be sure to do the KubeDirector teardown before deleting the GKE cluster! Otherwise, some dangling resources may remain in your gcloud project, especially those related to implementing LoadBalancer services. If you are in a development situation where "make teardown" doesn't work for some reason, then use individual "kubectl delete" invocations to delete as many resources as possible (reference how the Makefile implements the teardown action).

Once you have successfully completed the "make teardown" or done the individual resource deletions, you can delete your GKE cluster if you have no further use for it:
```bash
    gcloud container clusters delete my-gke
```

When you delete the GKE cluster, this will also delete the related context from your kubectl config. If you have some other context that you wish to return to using at this point, you will want to run "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config use-context" to select one.
