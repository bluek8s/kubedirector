#### KUBERNETES SETUP

If you intend to deploy KubeDirector on EKS, you will need to have AWS credentials. You must also have kubectl, and the aws CLI, and (for aws CLI versions before 1.16.156) the aws-iam-authenticator utility ready to use.

The [Getting Started with Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html) guide will walk you through all first-time setup as well as the process of creating a cluster. Both the AWS Management Console (web UI) process as well as the eksctl (command-line) process will work fine, but we recommend becoming familiar with the eksctl process if you will be repeatedly deploying EKS clusters.

Two important notes to be aware of when creating an EKS cluster:
* Be sure to specify Kubernetes version 1.14 or later.
* Choose a worker [instance type](https://aws.amazon.com/ec2/instance-types/) with enough resources to host at least one virtual cluster member. The example type t3.medium is probably too small; consider using t3.xlarge or an m5 instance type.

Use of eksctl and the AWS Management Console can be somewhat intermixed, because in the end they are just manipulating standard AWS resources, but this doc will assume you're just using one process or the other.

#### KUBECTL SETUP

In the AWS Management Console process, step 2 of [the guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-console.html) describes how to update your kubectl config using the aws CLI. The guide then walks you through using kubectl to add workers to the EKS cluster, so by the time you have a complete cluster you should definitely know that your kubectl is correctly configured.

In the eksctl process, your kubectl config will be automatically updated as a consequence of the EKS cluster creation.

In either case, kubectl will now access your EKS cluster as a member of the system:masters group that is granted the cluster-admin role.

#### DEPLOYING KUBEDIRECTOR

From here you can proceed to deploy KubeDirector as described in [quickstart.md](quickstart.md).

#### CONFIGURING KUBEDIRECTOR

After deploying KubeDirector but before creating virtual clusters, you may wish to create a KubeDirectorConfig object as described in [quickstart.md](quickstart.md).

This is particularly useful to address [an issue with storage classes](https://github.com/kubernetes/kubernetes/issues/34583) that is peculiar to EKS. In EKS, a storage class that will be used for container persistent storage must have its volumeBindingMode property set to the value "WaitForFirstConsumer". However, the "gp2" storage class that is the default in EKS clusters is not currently configured this way.

The volumeBindingMode property of an existing storage class cannot be modified, so to deal with this issue you must create another storage class and then either set it as the K8s default or else explicitly configure KubeDirector to use it.

A YAML file is available in the "deploy/example_configs" subdirectory to address this issue. It creates a storage class with the necessary property, and also creates a KubeDirectorConfig to direct KubeDirector to use that storage class. You can use kubectl to apply this solution:
```
    kubectl create -f deploy/example_configs/eks-gp2-for-kd.yaml
```

If you teardown and then re-deploy KubeDirector, you will need to repeat this step before using persistent storage.

#### WORKING WITH KUBEDIRECTOR

The process of creating and managing virtual clusters is described in [virtual-clusters.md](virtual-clusters.md).

#### TEARDOWN

When you're finished working with KubeDirector, you can tear down your KubeDirector deployment:
```bash
    make teardown
```

If you now want to completely delete your EKS cluster, you can.

If are using the AWS Management Console process, you should delete the cluster in the Amazon EKS console UI and delete the CloudFormation stack used to create the worker nodes. You can also delete the CloudFormation stack used to create the cluster VPC, or you can leave it for re-use with future clusters.

If you are using the eksctl process, the "eksctl delete cluster" command should clean up all resources it created.

The "eksctl delete cluster" command will also delete the related context from your kubectl config, but if you are using the AWS Management Console process you will need to do this cleanup yourself. You can use "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config delete-context" to remove the context associated with the deleted cluster.

If you have some other kubectl context that you wish to return to using at this point, you will want to run "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config use-context" to select one.
