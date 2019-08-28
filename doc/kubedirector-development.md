#### BASICS

First, make sure you have the prerequisite K8s setup described in [quickstart.md](quickstart.md).

The remainder of this document assumes that you have cloned the KubeDirector repo and that you are familiar with the concepts covered in [quickstart.md](quickstart.md).

Creating and managing virtual clusters with KubeDirector is described in [virtual-clusters.md](virtual-clusters.md).

#### DEVELOPMENT SETUP

If you intend to build KubeDirector yourself, rather than deploying a pre-built image, then some additional setup is required.

KubeDirector has been successfully built and deployed from macOS, Ubuntu, and CentOS. Similar OS environments may also work for development but have not been tested.

KubeDirector is written in the ["go"](https://golang.org/) language, so the fundamental requirement for building KubeDirector from source is to have that language installed (version 1.10 or later). The ["dep"](https://golang.github.io/dep/) tool is also required.

KubeDirector currently uses the [Operator SDK](https://github.com/operator-framework/operator-sdk) to do code generation for watching custom resources (the "informer" block in the [architecture diagrams](https://github.com/bluek8s/kubedirector/wiki/KubeDirector-Architecture-Overview)). So if you intend to build KubeDirector from source, you will need the operator SDK on your build system. Do the following step once before any build of KubeDirector:
```bash
    git clone https://github.com/operator-framework/operator-sdk.git $GOPATH/src/github.com/operator-framework/operator-sdk
    cd $GOPATH/src/github.com/operator-framework/operator-sdk
    git checkout v0.8.1
    make dep
    make install
```

Note the specific operator-sdk version that is used above; this will undoubtedly change in future KubeDirector versions.

You will also need Docker installed on your build system.

If you intend to share your KubeDirector image by pushing it to a Docker registry, then:
* Create a Local.mk file that sets the "image" variable to the name of your desired image. E.g. "image=my_dockerhub_repo/my_imagename:my_tag" or "image=quay.io/my_quay_repo/my_imagename:my_tag"
* Make sure your registry of choice is accessible from your K8s nodes, and that your local Docker credentials are set so that you are allowed to push to that registry.

If however you intend to experiment with KubeDirector builds but you do NOT intend to push your image to a registry, you can do that without needing to have a custom image name and registry credentials. You will still need to have Docker installed on your build system though. More about this below.

#### NOTES ON THE RED HAT UBI

We use the Red Hat Universal Base Image (UBI) as a base for building the KubeDirector image. The KubeDirector build process will usually transparently handle downloading the UBI from the appropriate Red Hat-managed repo as necessary.

However we have observed an issue with this process in CentOS development environments using an old version of Docker (e.g. 1.13). This issue manifests as the error message "open /etc/docker/certs.d/registry.access.redhat.com/redhat-ca.crt: no such file or directory" during an attempt to download the UBI.

If you do encounter this error when using an old version of Docker on CentOS, and if you are a sudo-privileged user, the following steps should resolve the issue:
```bash
    sudo rpm -e --nodeps subscription-manager-rhsm-certificates
    sudo yum --setopt=obsoletes=0 install -y python-rhsm-certificates
```

However the best solution is probably to move to using a more recent release of Docker Engine if that is possible.

#### BUILDING

Make sure that "$GOPATH/bin" is included in your PATH environment variable.

To build KubeDirector for the first time:
```bash
    make dep
    make build
```

When rebuilding KubeDirector subsequently, only "make build" should be necessary, unless you have changed the set of packages that the code imports.

The build process creates the YAML for the KubeDirector deployment, the kubedirector binary, and a "configcli" package of utility Python code. It then creates a Docker image that contains the kubedirector binary at /usr/local/bin/kubedirector, and has the configcli package stored at /root/configcli.tgz.

The Docker image will have some default name associated with the KubeDirector version (shown in the "make build" output), unless you have redefined the image name using a Local.mk file as described above.

Once you have built KubeDirector, any subsequent "make deploy" will use your locally generated deployment resource spec. To return to using the pre-built spec, do "make clean".

#### DEPLOYING

Whenever you do "make deploy", KubeDirector is deployed to K8s using the image indicated in the deployment resource spec. If you have built KubeDirector locally, the deployment spec will reference the Docker image name used during the most recent build.

A "make push" will push your locally built image to its registry, so that it can be deployed. If you have not set a custom image name, "make push" will fail.

If you *have* set a custom image name, then one possible clean/rigorous cycle of deploying successive builds would be:
1. "make build" (preceded by "make dep" if necessary)
2. "make push"
3. "make deploy"
4. testing
5. "make teardown"
6. make code changes
7. repeat from step 1

If you haven't set a custom image name and established credentials for pushing it to a remote registry, that flow will not work for you. Even if you have or could, however, you still may not wish to use that flow. It's somewhat tedious and it removes any existing virtual clusters (unless you follow a more elaborate sequence). Also, if you are not changing the image tag for each cycle, it can cause issues for anyone else using that same image.

A different option that is suitable for quick tests of intermediate builds is to use "make redeploy". This leaves all K8s resources in place, injecting your locally built kubedirector binary and configcli package into the existing KubeDirector deployment. (In a "make redeploy" the locally built Docker image is not used, but there is not currently an easy way to skip building it.)

Before starting a "redeploy" cycle, you do need an initial deployment. If you don't have a custom image name, this initial deployment will use a public KubeDirector image:
* "make deploy"

After the initial deploy, your development cycle can look like this:
1. "make build" (preceded by "make dep" if necessary)
2. "make redeploy"
3. testing
4. make code changes
5. repeat from step 1

If you have made changes that affect the RBAC or the KubeDirector deployment resource spec, you'll need to reset the cycle with a "make teardown" followed by "make deploy". Then you can immediately do "make redeploy" and start testing again.
