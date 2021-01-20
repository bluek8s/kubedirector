## Overview
The app is built with MapR 6.1 MEP 6.3 components.

## What is MapR?
MapR is a complete enterprise-grade distribution for Apache Hadoop. The MapR Converged Data Platform has been engineered to improve Hadoop’s reliability, performance, and ease of use. 
The MapR distribution provides a full Hadoop stack that includes the MapR File System (MapR-FS), the MapR-DB NoSQL database management system, MapR Streams, the MapR Control System (MCS) user interface, and a full family of Hadoop ecosystem projects. You can use MapR with Apache Hadoop, HDFS, and MapReduce APIs.

# Details: 

### Application Image Details:

#### Version:
1.0

#### Software Included
    Component	       Component Version
    =================================
    Apache Hadoop       2.7.0
    Apache ZooKeeper    3.4.5
    MapR Core           6.1
    Apache Oozie        5.1

#### Default login credential
    MapR admin user: mapr
    No password is set for this user by default.
    Note: any local user created by app will not have password set
    Note: User can pass the AD/LDAP user details by following the procedure to enable AD/LDAP on Kubedirector

# Details on the Kubedirector App and its features/facilities
MapR kubedirector app enables user to submit YARN jobs. By default following roles are defined in the app

     Roles	                Services running roles
    ==============================================================================
     control-system         On this pod/role MapR Control System, Zookeeper and 
                            YARN History Server are running. MapR Control System is
                            facility to manage MapR Ecosystem. It has a Web UI which
                            can be launched from "Service Endpoints" tab under
                            kubedirector's tenant -> Applications. Only one instance
                            of control-system is supported. To launch YARN History
                            server UI, follow procedure similar to launching MapR
                            Control System as mentioned earlier.
     cldb                   On this pod/role MapR CLDB, Zookeeper and MapR Fileserver
                            services are running. This pod serves the mapr tickets for
                            secure cluster. In secured cluster mapr tickets are necessary
                            to submit jobs. Only one instance of cldb is supported
     resource-manager       YARN Resource-Manager customized for MapR Eco system runs
                            on this pod/role along with Zookeeper. When more than one member 
                            is chosen for this role then resource-manager will be 
                            configured as HA. If more than one resource-manager is chosen
                            then Zookeeper will run only on the 1st pod.
     nodemanager            YARN Nodemanager service customized for MapR Eco system runs
                            on this pod/role along with MapR Fileserver. More than one 
                            nodemanager can be created based on the need.
     oozie                  Apache Oozie service customized for MapR Eco system runs on
                            this pod/role. Currently only one pod of oozie is supported.
     edge                   It provides ssh service and hive client services. So users
                            can connect this pod through ssh. More than one edge role
                            is allowed

# Sample Tests

#### Finding information to connect to pod
We can login pods of different roles through "kubectl exec" to respective pods. For "edge" role we can also do ssh. To do "kubectl exec" we need to know the pod name or to ssh we need to know the url and port this can be found under "Service Endpoints" tab of Kubedirector tenant applications.

#### Procedure to generate mapr ticket on secure cluster
To run jobs on secured MapR cluster, mapr ticket is mandatory. Switch to MapR admin user user using "su". Now to generate the mapr ticket run "maprlogin password" and provide MapR admin user's password when prompted. This will generate mapr ticket. If the user is local user and not AD/LDAP user 
then set the passwd for the local user first and then generate the ticket.


#### Running sample YARN job
To run sample YARN job, login to "edge" role either through "ssh" or through "kubectl exec". Generate MapR ticket as explained earlier for the user. Once ticket is generate run the below instruction to trigger TestDFSIO YARN job

`hadoop jar /opt/mapr/hadoop/hadoop-2.7.0/share/hadoop/mapreduce/hadoop-mapreduce-client-jobclient-2.7.0-mapr-1808-tests.jar TestDFSIO -write -nrFiles 2 -size 10`

In the above code i am running TestDFSIO write test. I am creating 2 file of size 10.
       
#### Docker image location

There are 2 images used in the app and the following are the location of images.

docker.io/bluedata/datafabric610:1.0
docker.io/bluedata/datafabric610edge:1.0