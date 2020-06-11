#### BASICS

First, make sure you have the prerequisite K8s setup described in [quickstart.md](quickstart.md).

The remainder of this document assumes that you have cloned the KubeDirector repo and that you are familiar with the concepts covered in [quickstart.md](quickstart.md).

Creating and managing virtual clusters with KubeDirector is described in [virtual-clusters.md](virtual-clusters.md).

#### CODEBASE MIGRATION

If you are working with a fresh checkout of this codebase, skip ahead to DEVELOPMENT SETUP below.

If however you are updating to a newer version of the KubeDirector source after working with an older version, it may not be desirable to do a fresh checkout if your codebase contains extensive changes. In that case, the following steps are recommended:
* If possible, do a "make clean" **and then** pull/merge in the changes for the new source. If you have already pulled in the new changes however, go ahead and do a "make clean" now.
* If you are coming from v0.4.0 (or earlier), recursively remove the "vendor" directory and its contents.
* Make sure there are no old generated source files remaining (generated files live in subdirectories under "pkg/apis"). "git status" can help you check for this; any "untracked" go source files should likely be removed unless you know exactly what they are and why they should remain.

Then proceed to the "DEVELOPMENT SETUP" section below and check for any new tool requirements.

#### DEVELOPMENT SETUP

If you intend to build KubeDirector yourself, rather than deploying a pre-built image, then some additional setup is required.

KubeDirector has been successfully built and deployed from macOS, Ubuntu, and CentOS. Similar OS environments may also work for development but have not been tested.

KubeDirector is written in the ["go"](https://golang.org/) language, so the fundamental requirement for building KubeDirector from source is to have that language installed (version 1.13 or later).

KubeDirector uses the [Operator SDK](https://github.com/operator-framework/operator-sdk) to do code generation for watching custom resources. The version of the Operator SDK used by KubeDirector depends on which release or branch of the KubeDirector source you are working with. So before you proceed, make sure that you are looking at the version of this document corresponding to the release/branch of KubeDirector that you care about! For example if you are currently working with some specific KubeDirector release on your local workstation, but you are reading this document from the tip of the master branch on GitHub, then you may end up with incorrect information.

KubeDirector currently uses version 0.15.2 of the Operator SDK. You should reference [that version of the Operator SDK installation guide](https://github.com/operator-framework/operator-sdk/blob/v0.15.2/doc/user/install-operator-sdk.md), and you should make sure that you specifically install version 0.15.2 of the operator-sdk tool. The most foolproof way to get the correct version is to use the ["Install from GitHub release"](https://github.com/operator-framework/operator-sdk/blob/v0.15.2/doc/user/install-operator-sdk.md#install-from-github-release) section of that doc.

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

#### NOTES ON GOROOT ISSUES

When using a precompiled operator-sdk binary to build an operator such as KubeDirector, you may encounter this error:
```
    operator-sdk generate k8s
    INFO[0000] Running deepcopy code-generation for Custom Resource group versions: [kubedirector:[v1beta1], ]
    F0327 12:51:51.104843   84262 deepcopy.go:885] Hit an unsupported type invalid type for invalid type, from github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1.KubeDirectorApp
    make: *** [pkg/apis/kubedirector/v1beta1/zz_generated.deepcopy.go] Error 255
```

This can be resolved by explicitly setting and exporting your GOROOT environment variable (if it is currently unset). You can find the necessary value by executing "go env GOROOT".

As an example of one way to tackle this issue that will be robust to future golang version upgrades... if you have the following two lines in one of your profile scripts (like .bash_profile or .bashrc):
```bash
    export GOPATH=~/Projects/go
    export PATH=$PATH:$GOPATH/bin
```
then immediately following those two lines you could add this one:
```bash
    export GOROOT=`go env GOROOT`
```

#### BUILDING

Make sure that "$GOPATH/bin" is included in your PATH environment variable.

To build KubeDirector for the first time:
```bash
    make build
```

If you subsequently make edits that change the set of packages that the code imports, you should run "make modules" before rebuilding.

The build process creates the YAML for the KubeDirector deployment, the kubedirector binary, and a "configcli" package of utility Python code. It then creates a Docker image that contains the kubedirector binary at /usr/local/bin/kubedirector, and has the configcli package stored at /root/configcli.tgz.

The Docker image will have some default name associated with the KubeDirector version (shown in the "make build" output), unless you have redefined the image name using a Local.mk file as described above.

Once you have built KubeDirector, any subsequent "make deploy" will use your locally generated deployment resource spec. To return to using the pre-built spec, do "make clean".

#### DEPLOYING

Whenever you do "make deploy", KubeDirector is deployed to K8s using the image indicated in the deployment resource spec. If you have built KubeDirector locally, the deployment spec will reference the Docker image name used during the most recent build.

A "make push" will push your locally built image to its registry, so that it can be deployed. If you have not set a custom image name, "make push" will fail.

If you *have* set a custom image name, then one possible clean/rigorous cycle of deploying successive builds would be:
1. "make build" (preceded by "make modules" if necessary)
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
1. "make build" (preceded by "make modules" if necessary)
2. "make redeploy"
3. testing
4. make code changes
5. repeat from step 1

If you have made changes that affect the RBAC or the KubeDirector deployment resource spec, you'll need to reset the cycle with a "make teardown" followed by "make deploy". Then you can immediately do "make redeploy" and start testing again.

Note: if you are using this redeploy cycle for your testing, you could choose to substitute "make compile" for "make build" in step 1. This will be faster because it only builds the KubeDirector executable, without rebuilding the container image. You will need to be sure to finally do "make build" before any "make push" however, because otherwise your container image will not be up-to-date with your tested changes.
