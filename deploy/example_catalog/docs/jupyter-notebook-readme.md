#### Jupyterhub BRIEF
[JupyterHub](https://jupyterhub.readthedocs.io/en/stable/getting-started/index.html) is the best way to serve Jupyter notebook for multiple users. It can be used in a classes of students, a corporate data science group or scientific research group. It is a multi-user Hub that spawns, manages, and proxies multiple instances of the single-user Jupyter notebook server.

#### Jupyterhub ROLES
* Jupyterhub notebook app has only one role - controller. Jupyterhub notebooks can be used by AD-LDAP users that are configured to be used on Kubernetes cluster. Version control on notebooks could be managed using Bit Bucket and Github. Jupyterhub notebooks are designed for developing and testing code and running machine learning training jobs on training cluster. Support for Python notebooks and R is available. Support for DataTap is available.

#### Support on Jupyterhub
* Jupyterhub notebooks can be used with or without source control but they have to be used by AD/LDAP users only.
* To create notebook cluster with LDAP secret, we should provide "hpecp-ext-auth-secret" in secrets section under connections in notebook_cluster.yaml. Note that for using Notebooks, this is mandatory. 
* To add source control support, we have to add source control details in Kubernetes AIML tenant UI. Copy the secret and paste the same in the notebook_cluster.yaml. Git hub repo should provide push, pull, and clone access to the user that has to use notebook cluster. Using source control is optional.
* To attach training clusters to notebook app, they have to be deployed first. Once they are deployed, their name should be added in "clusters" section under connections in notebook_cluster.yaml. We can have multiple training clusters attached to the notebook cluster. They can be accesssed using %attachments magic command from notebook. Attaching training clusters is optional.
* To connect to DTAP data store, we need to add podLabels sections. This is optional.
* Jupyterhub supports Python version 3.8 currently.

Note: Details of Dtap and all above supported connections - source control secret, ad-ldap secret, training app cluster is added in the [example](https://github.com/bluek8s/kubedirector/blob/master/deploy/example_clusters/cr-cluster-jupyter-notebook.yaml) for easy reference. 

#### QA Tests performed:
* Create a cluster with various AD / LDAP settings. 
* Create a cluster with or without source control support
* Create a cluster with or without training app being connected to notebook cluster.
* Create a cluster with or without Dtap support.
* Create a cluster. Open/Save existing python notebooks. Create new python notebooks to import existing packages, install new packages and run a complete AIML workflow
* Delete Notebook pod and verify Kubernates re-creates node. Notebook app can be worked upon after the pod is recreated automatically by Kubernetes. All saved older updates are available on new pod.

#### Docker Image location:
* docker.io/bluedata/kd-notebook:1.8

#### Docker Pull Command:
* docker pull docker.io/bluedata/kd-notebook:1.8
