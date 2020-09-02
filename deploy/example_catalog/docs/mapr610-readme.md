## Overview
The app is built with MapR 6.1 MEP 6.3 components.

## What is MapR ?
MapR is a complete enterprise-grade distribution for Apache Hadoop. The MapR Converged Data Platform has been engineered to improve Hadoop’s reliability, performance, and ease of use. 
The MapR distribution provides a full Hadoop stack that includes the MapR File System (MapR-FS), the MapR-DB NoSQL database management system, MapR Streams, the MapR Control System (MCS) user interface, and a full family of Hadoop ecosystem projects. You can use MapR with Apache Hadoop, HDFS, and MapReduce APIs.

# Details: 

### Application Image Details:

#### Version:
1.2

#### Software Included
    Component	       Component Version
    =================================
    Apache Hadoop       2.7.0
    HttpFS              1.0
    Apache Hive         2.3
    Hue                 4.3.0
    Apache ZooKeeper    3.4.5
    Tez                 0.9
    MapR Core           6.1

#### Default login credential
    MapR admin user: mapr
    MapR admin password: mapr (standard password unless modified)

# Details on the Kubedirector App and its features/facilities
MapR kubedirector app enables user to submit hive jobs on Tez. Hive jobs can be submitted either through hive client or Hue console. These Hive and Hue are customized to use the facilities offered by MapR-FS. By default following roles are defined in the app

     Roles	                Services running roles
    ==============================================================================
     control-system         On this pod/role MapR Control System is running. Its a
                            facility to manage MapR Ecosystem. It has a Web UI which
                            can be launched from "Service Endpoints" tab under
                            kubedirector's tenant -> Applications. Only one instance
                            of control-system is supported
     cldb                   This pod serves the mapr tickets for secure cluster.
                            In secured cluster mapr tickets are necessary to
                            submit jobs. Only one instance of cldb is supported
     zookeeper              Apache zookeeper customized for MapR Eco System is
                            running on this role. 3 instances of zookeeper runs
     resource-manager       YARN Resource-Manager customized for MapR Eco system runs
                            on this pod/role. When more than one member is chosen for
                            this role then resource-manager will be configured as HA
     history-server         YARN History Server customized for MapR Eco system runs
                            on this pod/role. Only one instance of the role is supported
     nodemanager            YARN Nodemanager service customized for MapR Eco system runs
                            on this pod/role. More than one nodemanager can be created
                            based on the need.
     fileserver             Exclusive pod/role for MapR's MFS (MapR Filesystem). This
                            service serves the storage need for MapR MFS. MapR MFS is
                            running on all roles except edge role. Exclusive fileserver
                            role is for dynamically update the cluster's storage need.
                            Adding fileserver roles increases the storage availability.
     hive-meta              Hive Metastore customized for MapR Eco system runs on this
                            pod/role. Hive Metastore keeps all the meta data in MySQL
                            database. When more than one member is chosen for this role,
                            it will be configure operate in a HA mode.
     hive-server2           Hive Server2 services customized for MapR Eco system runs on
                            this pod/role. Hive Server2 is necessary to run all the hive
                            queries. When more than one Hive Server2 is chosen, it will
                            operate in HA mode. Hive-server2 is configured to run on Tez
     hue                    Runs Apache Hue services customized for MapR Eco system. Hue
                            is configured to connect to Hive-Server2 and execute Hive jobs
                            on Tez. When Hove-server2 is configured as HA. It is necessary
                            to modify the hue.ini file to match the active hive-server2.
                            Currently no automatic failover facility is available hence
                            have to manually update hue.ini with active hive-server2 pod.
     edge                   It provides ssh service and hive client services. So users
                            can connect this pod through ssh. More than one edge role
                            is allowed

# Sample Tests

#### Finding information to connect to pod
We can login pods of different roles through "kubectl exec" to respective pods. For "edge" role we can also do ssh. To do "kubectl exec" we need to know the pod name or to ssh we need to know the url and port this can be found under "Service Endpoints" tab of Kubedirector tenant applications.

#### Procedure to generate mapr ticket on secure cluster
To run jobs on secured MapR cluster, mapr ticket is mandatory. Switch to mapr user using "su". Now to generate the mapr ticket run "maprlogin password" and provide mapr user password when prompted. This will generate mapr ticket. This step can be skipped on unsecure cluster.

#### Prior Steps for running sample jobs
Create a sample csv file like students.csv file having students id, students name and class. You can have this file locally on pod or on maprfs. To copy the file to maprfs, generate mapr ticket (secure cluster only) and use hadoop fs command. For example to copy the students.csv file to /tmp on maprfs run "hadoop fs -put students.csv /tmp". Successful copy can be confirmed by running "hadoop fs -ls /tmp"

#### Running sample hive jobs
To run sample hive jobs, login to "edge" role either through "ssh" or through "kubectl exec". If secured cluster then switch to mapr user and generate mapr ticket. As mapr user start hive service by running "hive" command. Once hive prompt is seen, try running following sample hive instructions
1. Creating new database

        hive> create db sample;

2. Creating students table on new database using students.csv file which is saved in maprfs

        hive> use sample;

        hive> create external table students (student_id string, student_name string, student_class int) row format delimited fields terminated by "," stored as textfile location "maprfs:///tmp";

3. Displaying students details studying in class 10

        hive> use sample;

        hive> select * from students where student_class = 10;

#### Running sample hive jobs through Hue
Launch the Hue Web UI using the mapr user credential. If the hive sample jobs were executed before attempting to run Hue based Hive job then "sample" database should be visible on Hue console adn when clicked on "sample" database we should be able to see the "students" table. If "students" table is not present then submit the hive instructions given under hive samples in Hue console's hive editor and execute.
1. List all the students details from the students table of sample database
On the Hive editor of Hue console, against the Database option click on dropdown and select "sample". Execute the following instructions
        
        select * from students;

#### Sample usage of HttpFS
HttpFS is used to list the content of maprfs. It can be used 2 ways, through browser or through curl commands.HttpFS
1. Sample program to list contents of maprfs:///user. Launch the HttpFS web UI from "Service Endpoints". The URL is not complete and so will not be able to see anything. To complete the URL update in the below format. For example, to list the content of maprfs:///user

        http(s)://<link given in the service endpoints:port as per service endpoints>/webhdfs/v1/user?op=LISTSTATUS&user=mapr

2. Sample curl command to perform similar operation as 1st sample
    
        curl -u mapr -i "http(s)://<link given in the service endpoints:port as per service endpoints>/webhdfs/v1/user?op=LISTSTATUS&user=mapr"
        
#### Docker image location

There are 2 images used in the app and the following are the location of images.

docker.io/bluedata/mapr610:1.3
docker.io/bluedata/mapr610edge:1.3