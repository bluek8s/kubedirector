#### Apache Spark-2.4.5 BRIEF
[Apache Spark](https://spark.apache.org/docs/2.4.5/) is a fast and general-purpose cluster computing system. It provides high-level APIs in Java, Scala, Python and R, and an optimized engine that supports general execution graphs. It also supports a rich set of higher-level tools including Spark SQL for SQL and structured data processing, MLlib for machine learning, GraphX for graph processing, and Spark Streaming.

#### Jupyterhub BRIEF
[JupyterHub](https://jupyterhub.readthedocs.io/en/stable/getting-started/index.html) is the best way to serve Jupyter notebook for multiple users. It can be used in a classes of students, a corporate data science group or scientific research group. It is a multi-user Hub that spawns, manages, and proxies multiple instances of the single-user Jupyter notebook server.

#### Livy BRIEF
[Livy](https://livy.apache.org) enables programmatic, fault-tolerant, multi-tenant submission of Spark jobs from web/mobile apps (no Spark client needed). So, multiple users can interact with your Spark cluster concurrently and reliably.

#### Sparkmagic BRIEF
[Sparkmagic](https://github.com/jupyter-incubator/sparkmagic) is a library of kernels that allows Jupyter notebooks to interact with Apache Spark through Apache Livy, which is a REST server for Spark. Spark and Apache Livy are installed when you create a cluster with JupyterHub.

#### Apache Spark 2.4.5 with Jupyterhub ROLES
* Apache Spark Master is the main component for Spark loads. 
  * Spark is a cluster with one Spark Master node and minimum of 1 Spark Worker node. 
  * With the increase in load, number of Spark Workers can be scaled up horizontally to be able to take more load.
* Livy node is used to submit jobs from REST API or from Jupyter notebook. In the cluster, one node is required for Livy.
* Jupyterhub is an optional component. If user wants to submit jobs to Spark master from Jupyterhub notebook though Livy server, Jupyterhub requires one node.

##### Examples
There are two ways to use sparkmagic. Head over to the examples section for a demonstration on how to use both models of execution.
1. Via the IPython kernel
The Sparkmagic library provides a %%spark magic that you can use to easily run code against a remote Spark cluster from a normal IPython notebook. See the Spark Magics on IPython sample notebook
2. Via the PySpark and Spark kernels
The sparkmagic library also provides a set of Scala and Python kernels that allow you to automatically connect to a remote Spark cluster, run code and SQL queries, manage your Livy server and Spark job configuration, and generate automatic visualizations. See Pyspark and Spark sample notebooks.
3. Sending local data to Spark Kernel

#### QA Tests performed:
* Create a cluster. 
* Increase Spark Worker nodes, test that the messages produced and consumed without issues. Confluent Control center should show a healthy cluster.
* Reduce Spark Worker nodes, test message transfer from client and cluster state in control center.
* Power off nodes to see that the cluster health from control center is good and messages are transferred without issues.
* Delete node and verify Kubernates re-creates node.
* Run PySpark, Scala Spark, and SparkR from Jupyter Notebook through Livy server [References](https://spark.apache.org/docs/latest/ml-guide.html)

