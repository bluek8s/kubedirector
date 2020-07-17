#### CONCEPTS

This doc assumes that you are familiar with the topics covered on the [KubeDirector wiki](https://github.com/bluek8s/kubedirector/wiki), especially the page on [app definition authoring](https://github.com/bluek8s/kubedirector/wiki/App-Definition-Authoring-for-KubeDirector). The definition of [KubeDirectorApp](https://github.com/bluek8s/kubedirector/wiki/KubeDirectorApp-Definition) will be a useful reference during the authoring process.

You should also be familiar with the process of [creating and managing virtual clusters with KubeDirector](virtual-clusters.md).

The "deploy/example_catalog" directory contains several KubeDirectorApp resources that are applied when you do "make deploy". These determine what kinds of virtual clusters can be deployed using KubeDirectorCluster resources. Each resource also identifies the Docker image(s) and app setup package(s) that it uses. Before authoring new app definitions, examine these current examples and the contents of each component. Currently the Cassandra example is the easiest non-trivial example to understand, with TensorFlow a close runner-up.

The simplest authoring task would involve making a modification to an existing image or setup package, and then making a modified KubeDirectorApp to reference the modified artifact (and possibly accomodate other roles or services). A modified version of an existing KubeDirectorApp should keep the same "distroID" value but have a new "version" and a new metadata name; currently there is not a more sophisticated framework for KubeDirectorApp versioning.

A more complex authoring task is to make an app definition from scratch. The KubeDirectorApp, image(s), and any app setup packages will need to be iteratively developed and tested together.

This doc discusses some common operations and example workflows for those tasks.

#### HOSTED COMPONENTS

The KubeDirectorApp resource is the only component that will be hosted by the K8s platform itself.

A Docker image must be hosted at a registry that is accessible to the K8s nodes, since K8s will need to pull that image in order to deploy containers.

An app setup package will usually be hosted on a webserver that is accessible to the container network, since a process within the container will download it. (The hosting and network-accessibility requirements for app setup packages are under discussion.) Alternately this package can reside on the Docker image.

Part of establishing a successful app definition authoring workflow is the ability to quickly revise these hosted components. For the app setup package in particular, S3 bucket hosting has proven useful. An app setup package stored on the Docker image is less amenable to quick revision. The examples later in this document will assume a web-hosted package.

#### REGISTERING THE KUBEDIRECTORAPP

Registering a new KubeDirectorApp resource will be an operation that occurs in most development workflows. It will certainly be required when starting the development of a new app definition from scratch. But even if your intent is to modify an existing definition, it is best to create a (differently-named) copy of the existing KubeDirectorApp and register that new resource as your working version, so as to avoid disturbing the existing definition.

Given a KubeDirectorApp resource "another_app.yaml" that points to valid image(s) and setup package(s), you can register this app type with KubeDirector using kubectl:
```bash
    kubectl create -f another_app.yaml
```

Note that it can matter which namespace you choose as the home for this resource. When a new KubeDirectorCluster is created and references some KubeDirectorApp name, KubeDirector first looks in the namespace of that KubeDirectorCluster to find the referenced KubeDirectorApp. If no KubeDirectorApp by that name is found there, then KubeDirector will next look in its own namespace.

#### MODIFYING THE KUBEDIRECTORAPP

Another common operation is to modify or even remove an existing KubeDirectorApp. In the iterative process of developing an app definition, it will often be useful to modify the KubeDirectorApp in-place so that you don't end up with a proliferation of slightly-differently-named KubeDirectorApp resources all registered with KubeDirector.

To avoid unexpected consequences however, don't do this if any existing KubeDirectorCluster resources are referencing this app type. In the future, such an action will be explicitly blocked.

Assuming there are no existing virtual clusters referencing it, you could modify the KubeDirectorApp resource file "another_app.yaml" and then apply those changes with kubectl:
```bash
    kubectl apply -f another_app.yaml
```

If you need to delete an unreferenced KubeDirectorApp resource, you can do that with kubectl as well:
```bash
    kubectl delete -f another_app.yaml
```

#### MODIFYING AN IMAGE OR SETUP PACKAGE

If you modify a Docker image or an app setup package "in place" -- i.e., you make changes and then upload the new artifact back to its hosting without changing its name -- then no changes to the KubeDirectorApp resource are needed. Future KubeDirectorCluster deployments that reference that KubeDirectorApp will use the new image or setup package.

Such in-place modifications may or may not be a good idea. On the plus side, they are quick and easy. However, they can cause confusion if any existing virtual clusters are using the artifact, or even if any other KubeDirectorApp resources (besides the one you are working on) reference it. Therefore in-place modifications of an artifact are only safe if the artifact was newly named specifically for your current project.

In the case where you can't do in-place modification of an artifact, you therefore need to give it a new name when uploading your revised version. This also means that you will need to modify the KubeDirectorApp resource to point to this new name.

#### EXAMPLE: BEGINNING A NEW APP DEFINITION

1. Decide which app software should be installed in each role.
2. Decide which service endpoints should be available in each role.
3. Create a Dockerfile (either for all roles or per-role) to install the necessary software in the image.
4. Build the Docker image(s) and push them to your registry.
5. Create an app setup package (either for all roles or per-role). Can be a skeleton to start with.
6. Upload the setup package(s) to your web hosting.
7. Create a KubeDirectorApp resource to describe the roles and services, and to reference the image(s) and setup package(s).
8. Register the KubeDirectorApp with KubeDirector, using "kubectl create".
9. Go to the [Iterative Development](#example-iterative-development) example below.

#### EXAMPLE: MODIFYING AN EXISTING APP DEFINITION

To be conservative, this example involves renaming every image or setup package used by the app. In many cases this will not be necessary (when you are not modifying those components).

1. Copy the YAML for the existing KubeDirectorApp to a new file.
2. In that new file, modify the metadata name to be unique. Also modify the value of the "version" property.
3. For each image used by this app:
    1. Push the current image to your registry under a new name. If you have the image currently loaded into Docker, an easy way would be to use "docker tag" to give it a new tag (and perhaps new repo/name), then push that new tag.
    2. Modify your new KubeDirectorApp YAML to replace the old image name with the new name.
4. For each setup package used by this app:
    1. Upload the current setup package to your web hosting under a new name.
    2. Modify your new KubeDirectorApp YAML to replace the old package name with the new name.
5. Register the modified KubeDirectorApp with KubeDirector, using "kubectl create".
6. Go to the [Iterative Development](#example-iterative-development) example below.

#### EXAMPLE: ITERATIVE DEVELOPMENT

At this point in the example flows, you have a KubeDirectorApp registered under a new name, which no KubeDirectorCluster is using yet. This KubeDirectorApp references image(s) and setup package(s) that are not referenced by any other KubeDirectorApp, so you can do "in-place" component updates. Now you just need to get it working!

1. Create a KubeDirectorCluster resource that references your KubeDirectorApp.
2. Register the KubeDirectorCluster with KubeDirector, using "kubectl create".
3. Watch the progress of virtual cluster creation through the resource's status object, and/or through the KubeDirector logs, until the cluster is in a stable state.
4. Test the service endpoints that the virtual cluster should provide.
5. Use ssh (if supported in the app) and/or "kubectl exec" to enter the app containers and examine their state.
6. Modify images and/or setup packages as necessary.
7. Upload modified components to their hosting, doing "in-place" replacement.
8. Remove the deployed KubeDirectorCluster using "kubectl delete".
9. Repeat from step 2 (or step 1 if you want to change the cluster spec) until the virtual cluster is deployed successfully and functionally.

Once initial creation is working, you should also test the addition, removal, expand, and shrink of roles -- which will result in the addition or deletion of virtual cluster members. Depending on your KubeDirectorApp role definitions, some of these operations may not be allowed for some roles. For example a role with cardinality "1" must exist in the initial deployment and cannot be removed, shrunk, or expanded. A role with cardinality "2+" must exist initially and cannot be removed, or shrunk below 2 nodes. A role with cardinality "0+" does not need to be in the initial deployment and allows any of the operations.

The process of testing these resize operations is similar to iterating through the steps above, with an added phase (between steps 3 and 4) of
* Modify your KubeDirectorCluster YAML file to change role member counts or add/remove roles.
* Apply the change with "kubectl apply".
* Wait for the cluster to become stable again.

You may want to do multiple resize tests in a row, without deleting and redeploying the KubeDirectorCluster, for as long as the resizes continue to be successful. Once you encounter a problem however it is best to start over with a freshly deployed virtual cluster.

#### EXAMPLE: FINALIZING AN APP DEFINITION

At this point you have a KubeDirectorApp resource, image(s), and setup package(s) that work together to provide functional virtual cluster deployments. Depending on your naming strategies and release processes, you may or may not be done at this point -- perhaps you can just give the KubeDirectorApp resource to whoever else needs to use it.

If you need to move images or setup packages to some other hosting at this point, that will involve one final modification of the KubeDirectorApp to reference those locations. You may also wish to change the KubeDirectorApp name or version string. After those KubeDirectorApp modifications you should run one more testing pass using the final app definition, through the steps that by now will be familiar.

If you have changed the KubeDirectorApp name for release, you can use "kubectl delete" to remove the old development version of the KubeDirectorApp. Any virtual clusters referencing that version should have already been deleted at the end of the testing process.
