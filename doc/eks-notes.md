#### KUBERNETES SETUP

If you intend to deploy KubeDirector on EKS, you will need to have AWS credentials. You must also have kubectl, and the aws CLI, and (for aws CLI versions before 1.16.156) the aws-iam-authenticator utility ready to use.

The [Getting Started with Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html) guide will walk you through all first-time setup as well as the process of creating a cluster. Both the AWS Management Console (web UI) process as well as the eksctl (command-line) process will work fine, but we recommend becoming familiar with the eksctl process if you will be repeatedly deploying EKS clusters.

As part of this process you will have a choice whether or not to use "AWS Fargate". For example, in the eksctl docs the cluster creation section has two tabs "AWS Fargate-only cluster" and "Cluster with Linux-only workloads". You may wish to follow the available links to read more about Fargate. FWIW we do *not* yet use Fargate when testing KubeDirector deployment and any EKS-related docs in this repo are currently written in the context of a non-Fargate deployment.

Two other important notes to be aware of when creating an EKS cluster:
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

In older versions of EKS it was necessary to create a particular KubeDirector config object if you intended to use persistent storage. That seems to no longer be necessary, but if you are experiencing issues with persistent volumes then you may want to refer to the content of this section in an [older version of this doc](https://github.com/bluek8s/kubedirector/blob/v0.3.0/doc/eks-notes.md).

#### WORKING WITH KUBEDIRECTOR

The process of creating and managing virtual clusters is described in [virtual-clusters.md](virtual-clusters.md).

#### TEARDOWN

When you're finished working with KubeDirector, you can tear down your KubeDirector deployment:
```bash
    make teardown
```

If you now want to completely delete your EKS cluster, you can.

If are using the AWS Management Console process, you should delete the cluster in the Amazon EKS console UI and delete the CloudFormation stack used to create the worker nodes. You can also delete the CloudFormation stack used to create the cluster VPC, or you can leave it for re-use with future clusters.

If you are using the eksctl process, the "eksctl delete cluster" command should clean up all resources it created. Note that doing this immediately after deleting LoadBalancer-type services may fail with an error about "cannot delete orphan ELB Security Groups"; wait a few minutes and try again.

The "eksctl delete cluster" command will also delete the related context from your kubectl config, but if you are using the AWS Management Console process you will need to do this cleanup yourself. You can use "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config delete-context" to remove the context associated with the deleted cluster.

If you have some other kubectl context that you wish to return to using at this point, you will want to run "kubectl config get-contexts" to see which contexts exist, and then use "kubectl config use-context" to select one.
