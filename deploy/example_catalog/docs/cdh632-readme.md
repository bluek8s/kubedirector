#### CDH BRIEF
CDH is a  popular distribution of Apache Hadoop and related projects. CDH delivers the core elements of Hadoop – scalable storage and distributed computing – along with a Web-based user interface and vital enterprise capabilities.CDH has following mandatory roles:

* cmserver
* primary
* hive
* worker

and following optional roles

* gateway
* hbase
* impala
* apps
* standby(Applicable when cluster is in high availability mode)
* aribiter(Applicable when cluster is in high availability mode)

More about the CDH can be read [here.](https://docs.cloudera.com/documentation/enterprise/6/6.3/topics/cdh_intro.html)

#### CDH ROLES

* cmserver role runs cloudera manager service. 
* primary role runs namenode,resource manager,zookeeper,spark history server,job history server services. In case of high availability it also runs journal node and fail over controller.
* hive role runs Hive server service.
* worker role runs datanode,node manager, hbase region server(if hbase is selected), 
  impala daemon(if  impala is selected)
* gateway runs hdfs gateway service,yarn gateway service, spark gateway service, hive gateway service
* hbase role runs HBASE thrift server, HBASE Host Monitor and HBASE Rest Server.
* impala role runs Impala Statestore and Impala Catalog Server.
* apps role runs HUE Server, HUE Load Balancer(in high availability mode) and oozie server.
* standby role runs Standby Namenode, Standby Resource manager,Failover controller,zookeeper and journal
  node.
* arbiter role runs journal manager, job history server, zookeeper


#### ENABLING HIGH AVAILABILITY

High availability can be enabled by uncommenting configmap block and connections block in <b>deploy/example_clusters/cr-cluster-cdh632cm-stor.yaml</b>
  

#### STEPS FOR SAMPLE TEST RUNS

<p align="center"><b><u>HDFS Service</b></u></p>

 

 1. ***Calculating Pi***
     
    Login to any pod and run
    
      sudo -u hdfs hadoop jar /opt/cloudera/parcels/CDH/lib/hadoop-mapreduce/hadoop-mapreduce-examples.jar pi 10 100
  
 2.  ***DFSIO write***
  
     Login to any pod and run 
     
      sudo -u hdfs hadoop jar /opt/cloudera/parcels/CDH/lib/hadoop-mapreduce/hadoop-mapreduce-client-jobclient-3.0.0-cdh6.3.2-tests.jar TestDFSIO -write -nrFiles 10 -fileSize 1000 -resFile /tmp/TestDFSIOwrite.txt
      
 3. ***DFSIO read***
    
    Login to any pod and run

      sudo -u hdfs hadoop jar /opt/cloudera/parcels/CDH/lib/hadoop-mapreduce/hadoop-mapreduce-client-jobclient-3.0.0-cdh6.3.2-tests.jar TestDFSIO -read -nrFiles 10 -fileSize 1000 -resFile /tmp/TestDFSIOread.txt
 
<p align="center"><b><u>Spark Service</b></u></p>

 1.  ***Spark Pi Job***
  
       Login to any pod and run
       
       spark-submit --class org.apache.spark.examples.SparkPi  --master yarn-client  
       --num-executors 1 --driver-memory 512m  --executor-memory 512m --executor-cores 1    
        /opt/cloudera/parcels/CDH/lib/spark/examples/jars/spark-examples_2.11-2.4.0-
        cdh6.3.2.jar 10
        
2.   **PySpark wordcount job**
   
        a) Create a dir /test in HDFS

        sudo -u hdfs hdfs dfs -mkdir /test

        b) Create a file wc.txt and place it inside /test
        
        sudo -u hdfs hdfs dfs -put wc.txt /test/wc.txt
        
        c) Start PySpark shell
        
        sudo -u spark pyspark

        d) In PySpark shell,
                
        myfile = sc.textFile("/test/wc.txt")
        counts = myfile.flatMap(lambda line: line.split(" ")).map(lambda word: (word, 
        1)).reduceByKey(lambda v1,v2: v1 + v2)
        print counts.collect()

<p align="center"><b><u>HIVE Service</b></u></p>

  Login to hue and go to HIVE Browser. Execute HIVE queries

<p align="center"><b><u>HBASE Service</b></u></b></p>

   Go to HBASE Browser in Hue and execute HBASE queries.

#### ADDING WORKERS

   Workers can be added by increasing the number of members corresponding to the worker role in cluster yaml.

After changing the member count of  workers and reapplying the cluster,check if all the services are up after logging into cloudera manager.

#### DELETING WORKERS

Workers can be deleted by decreasing the number of members corresponding to the worker role in cluster yaml.

After changing the member count of  the workers and reapplying the cluster,check if all the services are up after logging into cloudera manager.


#### FUTURE WORK

Adding Kerberos support


#### Docker Image location

* docker.io/bluedata/cdh632multi

#### Docker Pull command

* docker pull docker.io/bluedata/cdh632multi:1.4

