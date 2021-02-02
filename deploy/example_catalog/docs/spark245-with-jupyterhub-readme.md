# Spark245 with Jupyterlab Kubedirector Application
Overview
Spark 2.4.5 with Jupyterhub for ML Ops is an application image which is used to submit jobs related to spark applications, Currently supported applications are python, pyspark, spark scala and SparkR

# Details:
## Application Image Details:

### AppName: 
HPE Ezmeral Spark 2.4.5 + Jupyterhub
### DistroId:
bluedata/spark245
### Version: 
5.2.0.35

### Cluster Type: 
Spark 2.4.5 with Jupyterhub

### Software Included: 
- CentOS 7
- JDK 8
- Spark 2.4.5
- Livy 0.7
- Python 3.8.3
- R with R packages (devtools, knitr, tidyr, ggplot2, shiny, sparklyr, IRkernel)
- Miniconda
- Jupyter Notebook
- JupyterHub
- sparkmagic
- nodejs
- sparkling-water
- HDP


## Deployment

To deploy the Spark 245 with Livy server and Jupyterhub notebook, it is required to follow the steps mentioned below,


Cluster yaml file is required to be updated to have dtap, External Auth support (LDAP or Active Directory) or default User to login to Jupyterhub

### Prerequisites:

If Active Directory or LDAP  details are required in Spark245 application for login credentials, Ensure the Kubernetes cluster is updated with required Active Directory or LDAP details and external group details and also check the "enable packaged Apps" option. 

In Kubernetes master, execute below commands to add label to a secret hpecp-ext-auth-secret
1. kubectl config set-context --current --namespace=hpecp 
2. kubectl label secrets hpecp-ext-auth-secret "kubernetes.hpe.com/resource-tenant-visibility"="True" 
3. After that, create a new Tenant to deploy Spark245 application

In a Tenant, To enable ActiveDirectory/LDAP details in Spark245 application, it is required to update, “hpecp-ext-auth-secret” in cluster yaml file during launch.

Add below line in secret section and uncomment the lines
  
    connections:
    secrets:
    - hpecp-ext-auth-secret

If end user specifies default username and password, that credential can be updated in launch yaml file 
	update below mentioned lines in yaml file at the beginning,

    apiVersion: v1
    kind: Secret
    metadata:
      name: notebookusersecret
      labels:
        kubedirector.hpe.com/secretType : notebookusersecret
    type: Opaque
    data:
      notebook_username: YWRtaW4K
      notebook_password: YWRtaW4xMjMK

Update username and password in notebook_username & notebook_password in the form of base64 encoded.
Update “notebookusersecret” in secret section and uncomment the same.
  
    connections:
      secrets:
      - notebookusersecret
    
If there is no Active Directory/LDAP details updated in Kubernetes cluster or username password is not specified in yaml file, default username and password to login jupyterhub is admin/admin123

Enable DTAP:
In the cluster yaml, add “podLabels” details as mentioned below, in Roles section,

   
     roles:
     - 
      id: "spark-master"
      members: 1
      resources: 
        requests: 
          memory: "4Gi"
          cpu: "2"
        limits: 
          memory: "4Gi"
          cpu: "2"
      storage: 
        size: "50Gi"
      podLabels:
           hpecp.hpe.com/dtap: "inject" 
---

### Deploy Spark245 Application
In a tenant, Application section, after above updates as part of Prerequisites, launch “Spark 2.4.5 + Jupyterhub” in a created Tenant. 
Once application is deployed, status is updated with “configured”
In Service endpoints, find all URL links to the modules, **Jupyterhub, Livy Web UI, Spark Master UI and Spark Worker**.

#### How to use Application:

**Jupyterhub:**

From the Service endpoints Tab, Select access URL for Jupyterhub, this leads to Jupyterhub login page.

<img width="1435" alt="end-points" src="https://user-images.githubusercontent.com/59432587/98362036-968a8680-2052-11eb-9cbe-c2518c704961.png">


Enter the username password, as per the configuration done in Kube-director cluster or _Cluster yaml_ file during deployment.
    Enter LDAP/AD username and password if LDAP/AD is configured
    Enter user defined username and password if user has defined in yaml file.
    Enter admin/admin123 if no credentials has been configured.

After successful login, it will take the user to Jupyterlab Home page. 

<img width="905" alt="jupyterlab-home" src="https://user-images.githubusercontent.com/59432587/98353915-05adae00-2046-11eb-9c51-d2d2f4904387.png">


In the page, option to select Python3/Pyspark/Spark/SparkR/Terminal screens.

When user selects,
**Python 3**,
 which leads to Notebook, where you can execute python program and shell command along with HDFS commands with DTAP (if dtap is configured).

On the cell enter command,

	!id 

and run. It displays the user details, as mentioned in the picture.
<img width="1436" alt="python3-commands" src="https://user-images.githubusercontent.com/59432587/98354488-c7fd5500-2046-11eb-9af2-f30bbfd0117d.png">

Also, exeute other commands, like !whoami, !which python, 
HDFS and hadoop commands as shown in the image.

Python 3 notebook can also be used to execute Python programs and install additional packages
<img width="1436" alt="python-install" src="https://user-images.githubusercontent.com/59432587/98359830-cd5e9d80-204e-11eb-9189-5d10001d44d1.png">

**Pyspark**
 This notebook can be used to execute Pyspark programs, where this Pyspark programs directs to Spark Master through Livy server.
 Sample program is as shown below,
 <img width="1430" alt="pyspark-note" src="https://user-images.githubusercontent.com/59432587/98356286-5a065d00-2049-11eb-89ba-6309abef1673.png">
 
 In the sample word count program, input file is taken from dtap storage and output is saved to dtap stage.
 <img width="618" alt="pyspark-output" src="https://user-images.githubusercontent.com/59432587/98356403-8a4dfb80-2049-11eb-90dc-b2e7b9c14f52.png">
 
 Similary, user can execute Machinne learning Pyspark programs.
 
 **Livy Server**
 Livy updates the session and programs execution status in the livy web page. 
 <img width="1437" alt="livy" src="https://user-images.githubusercontent.com/59432587/98357087-ab631c00-204a-11eb-974e-d381d4933177.png">
 
 
 **Spark Master**
 Details on Spark master web UI, 
 <img width="1437" alt="spark-master" src="https://user-images.githubusercontent.com/59432587/98357200-df3e4180-204a-11eb-9519-46b0e31a9c36.png">
 
 
 **Spark Workers**  
 Details on Spark Worker web UI,
 <img width="1438" alt="spark-worker" src="https://user-images.githubusercontent.com/59432587/98357262-f7ae5c00-204a-11eb-94c3-c6c72e469a7e.png">
 
 
 **Sparkmagic commands**
 User can execute sparkmagic commands on Pyspark SparkR and Spark - scala notebooks.
 Few sparkmagic commands are executed, 
 <img width="1434" alt="sparkmagic" src="https://user-images.githubusercontent.com/59432587/98359168-c6835b00-204d-11eb-94bb-0871a79aa450.png">
 
 Refer https://github.com/jupyter-incubator/sparkmagic for more details on sparkmagic commands.
 
 
 **SparkR**
 
 Using SparkR notebook, sample exeution of k-mean model program using R.
 <img width="1436" alt="sparkr" src="https://user-images.githubusercontent.com/59432587/98360865-98534a80-2050-11eb-8edc-7d45f2c5271f.png">
 
 
