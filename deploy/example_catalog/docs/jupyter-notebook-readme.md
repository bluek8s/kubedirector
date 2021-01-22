### Jupyterhub BRIEF
[JupyterHub](https://jupyterhub.readthedocs.io/en/stable/getting-started/index.html) is the best way to serve Jupyter notebook for multiple users. It can be used in a classes of students, a corporate data science group or scientific research group. It is a multi-user Hub that spawns, manages, and proxies multiple instances of the single-user Jupyter notebook server.

### Jupyterhub ROLES
* Jupyterhub notebook app has only one role - controller. Jupyterhub notebooks can be used by AD-LDAP users that are configured to be used on Kubernetes cluster. Version control on notebooks could be managed using Bit Bucket and Github. Jupyterhub notebooks are designed for developing and testing code and running machine learning training jobs on training cluster. Support for Python notebooks and R is available. Support for DataTap is available.

### Mandatory requirement :
Notebook cluster is required to be used with AD-LDAP users. To add AD-LDAP users, we need to do the following:
1. Create Kubedirector cluster with
    1. LDAP/AD details
    2. Select check box “enable packaged Apps”
2. In Kubernetes master, execute below commands to add label to a secret hpecp-ext-auth-secret
    1. kubectl config set-context --current --namespace=hpecp
    2. kubectl label secrets hpecp-ext-auth-secret "kubernetes.hpe.com/resource-tenant-visibility"="True"
3. Create Tenant, configure other desired resources/clusters like source control, training cluster. Update cluster yaml while launching Notebook cluster as detailed in Jupyterhub cluster configuration section.

### Notes and support on Jupyterhub:
1. Jupyterhub service runs as a non-root user.
2. Only AD-LDAP users are allowed access to any other AD/LDAP users home directory.
3. KD Clusters used to create AIML tenant are required to have AD/LDAP configuration and AIML tenant should be created only after meeting the Mandatory requirement steps 1 and 2.
4. First time login for a user with source control configured takes some time as git clone for the repository is performed. Henceforth, login in should be faster.
5. Jupyterhub notebook supports Python version 3.8 currently.

### Jupyterhub cluster configuration:
Note: 
* Jupyterhub notebooks can be used with or without source control but they have to be used by AD/LDAP users only.
* Details of Dtap and all above supported connections - source control secret, ad-ldap secret, training app cluster is added in the [example](https://github.com/bluek8s/kubedirector/blob/master/deploy/example_clusters/cr-cluster-jupyter-notebook.yaml) for easy reference. 

#### Attaching a training cluster
To attach training clusters to notebook app, they have to be deployed first. Once they are deployed, their name should be added in "clusters" section under connections in spec:connections:clusters section like below while creating notebook cluster app. We can have multiple training clusters attached to the notebook cluster. They can be accessed using %attachments magic command from notebook. Attaching training clusters is optional.

```
---
apiVersion: "kubedirector.hpe.com/v1beta1"
kind: "KubeDirectorCluster"
metadata:
  name: "jupyter-notebook"
spec:
  app: "jupyter-notebook"
  appCatalog: "local"
  connections:
    secrets:
      - "hpecp-ext-auth-secret"
      - "hpecp-source-control-secret-d4c2c7467201788666a6347ce339fc41"
    clusters:
      - "training-cluster"
  roles:
    -
      id: "controller"
      members: 1
      resources:
        requests:
          memory: "2Gi"
          cpu: "2"
        limits:
          memory: "2Gi"
          cpu: "2"

---
```
Note that the name `training-cluster` is the name that was given in this specific setup for the training cluster and it was deployed prior to launching notebook cluster.


#### Configuring Source Control:
To add source control support, we have to add source control details in Kubernetes AIML tenant UI. Then login as a tenant user, add user specific details for source control. Copy the secret and paste the same while creating notebook cluster in spec:connections:secret section. Token should provide push, pull, and clone access to the user that has to use notebook cluster. Using source control is optional.

```
---
apiVersion: "kubedirector.hpe.com/v1beta1"
kind: "KubeDirectorCluster"
metadata:
  name: "jupyter-notebook"
spec:
  app: "jupyter-notebook"
  appCatalog: "local"
  connections:
    secrets:
      - "hpecp-ext-auth-secret"
      - "hpecp-source-control-secret-d4c2c7467201788666a6347ce339fc41"
    clusters:
      - "training-cluster"
  roles:
    -
      id: "controller"
      members: 1
      resources:
        requests:
          memory: "2Gi"
          cpu: "2"
        limits:
          memory: "2Gi"
          cpu: "2"
---
```
Note that in the below example `hpecp-source-control-secret-d4c2c7467201788666a6347ce339fc41` is the source control secret that was copied from source control page

#### Configuring DTAP :
To connect to DTAP data store, we need to add the same under podLabels in cluster specification as given below. This is optional.
```
---
apiVersion: "kubedirector.hpe.com/v1beta1"
kind: "KubeDirectorCluster"
metadata:
  name: "jupyter-notebook"
spec:
  app: "jupyter-notebook"
  appCatalog: "local"
  connections:
    secrets:
      - "hpecp-ext-auth-secret"
      - "hpecp-source-control-secret-d4c2c7467201788666a6347ce339fc41"
    clusters:
      - "training-cluster"
  roles:
    -
      id: "controller"
      members: 1
      resources:
        requests:
          memory: "2Gi"
          cpu: "2"
        limits:
          memory: "2Gi"
          cpu: "2"
      podLabels:
        hpecp.hpe.com/dtap: "inject"
---
```

#### Configuring AD/LDAP :

To use notebook app cluster with AD-LDAP users, we need to perform `Mandatory requirements` as specified above and add the secret “hpecp-ext-auth-secret” in spec:connections:secret section in the cluster specification.

```
---
apiVersion: "kubedirector.hpe.com/v1beta1"
kind: "KubeDirectorCluster"
metadata:
  name: "jupyter-notebook"
spec:
  app: "jupyter-notebook"
  appCatalog: "local"
  connections:
    secrets:
      - "hpecp-ext-auth-secret"
      - "hpecp-source-control-secret-d4c2c7467201788666a6347ce339fc41"
    clusters:
      - "training-cluster"
  roles:
    -
      id: "controller"
      members: 1
      resources:
        requests:
          memory: "2Gi"
          cpu: "2"
        limits:
          memory: "2Gi"
          cpu: "2"

---
```

### QA Tests performed:
* Create a cluster with various AD / LDAP settings. 
* Create a cluster with or without source control support
* Create a cluster with or without training app being connected to notebook cluster.
* Create a cluster with or without Dtap support.
* Create a cluster. Open/Save existing python notebooks. Create new python notebooks to import existing packages, install new packages and run a complete AIML workflow

### Docker Image location:
* docker.io/bluedata/kd-notebook:1.10

### Docker Pull Command:
* docker pull docker.io/bluedata/kd-notebook:1.10
