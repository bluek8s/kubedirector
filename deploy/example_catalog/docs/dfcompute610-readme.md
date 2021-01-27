## Overview
The app is built with Datafabric Compute 6.1 MEP 6.3 components.

## What is Datafabric Compute?
Datafabric Compute is a complete enterprise-grade distribution for Apache Hadoop. The Datafabric Compute Converged Data Platform has been engineered to improve Hadoop’s reliability, performance, and ease of use. 
The Datafabric Compute distribution provides a full Hadoop stack that includes the Datafabric File System (MapR-FS), the Datafabric-DB (MapR-DB) NoSQL database management system, Datafabric Compute Streams, the Datafabric Compute Control System (MCS) user interface, and a full family of Hadoop ecosystem projects. You can use Datafabric Compute with Apache Hadoop, HDFS, and MapReduce APIs.

# Details: 

### Application Image Details:

#### Version:
1.0

#### Software Included
    Component	       Component Version
    =================================
    Apache Hadoop       2.7.0
    Apache ZooKeeper    3.4.5
    Datafabric Core     6.1
    Apache Oozie        5.1

#### Default login credential
    Datafabric Compute admin user: mapr
    No password is set for this user by default.
    Note: any local user created by app will not have password set
    Note: User can pass the AD/LDAP user details by following the procedure to enable AD/LDAP on Kubedirector

# Details on the Kubedirector App and its features/facilities
Datafabric Compute kubedirector app enables user to submit YARN jobs. By default following roles are defined in the app

     Roles	                Services running roles
    ==============================================================================
     control-system         On this pod/role Datafabric Compute Control System, Zookeeper and 
                            YARN History Server are running. Datafabric Compute Control System is
                            facility to manage Datafabric Compute Ecosystem. It has a Web UI which
                            can be launched from "Service Endpoints" tab under
                            kubedirector's tenant -> Applications. Only one instance
                            of control-system is supported. To launch YARN History
                            server UI, follow procedure similar to launching Datafabric Compute 
                            Control System as mentioned earlier.
     cldb                   On this pod/role Datafabric CLDB, Zookeeper and Datafabric 
                            Fileserver services are running. This pod serves the tickets for
                            secure cluster. In secured cluster tickets are necessary
                            to submit jobs. Only one instance of cldb is supported
     resource-manager       YARN Resource-Manager customized for Datafabric Compute Eco system runs
                            on this pod/role along with Zookeeper. When more than one member 
                            is chosen for this role then resource-manager will be 
                            configured as HA. If more than one resource-manager is chosen
                            then Zookeeper will run only on the 1st pod.
     nodemanager            YARN Nodemanager service customized for Datafabric Compute Eco system 
                            runs on this pod/role along with Datafabric Fileserver. More than one 
                            nodemanager can be created based on the need.
     oozie                  Apache Oozie service customized for Datafabric ComputeEco system runs
                            on this pod/role. Currently only one pod of oozie is supported. It
     edge                   provides ssh service and hive client services. So users can connect
                            this pod through ssh. More than one edge role is allowed.

# Sample Tests

#### Finding information to connect to pod
We can login pods of different roles through "kubectl exec" to respective pods. For "edge" role we can also do ssh. To do "kubectl exec" we need to know the pod name or to ssh we need to know the url and port this can be found under "Service Endpoints" tab of Kubedirector tenant applications.

#### Procedure to generate ticket on secure cluster
To run jobs on secured Datafabric Compute cluster, ticket is mandatory. Switch to Datafabric Compute admin user user using "su". Now to generate the ticket run "maprlogin password" and provide Datafabric Compute admin user's password when prompted. This will generate ticket. If the user is local user and not AD/LDAP user 
then set the passwd for the local user first and then generate the ticket.


#### Running sample YARN job
To run sample YARN job, login to "edge" role either through "ssh" or through "kubectl exec". Generate ticket as explained earlier for the user. Once ticket is generate run the below instruction to trigger TestDFSIO YARN job

`hadoop jar /opt/mapr/hadoop/hadoop-2.7.0/share/hadoop/mapreduce/hadoop-mapreduce-client-jobclient-2.7.0-mapr-1808-tests.jar TestDFSIO -write -nrFiles 2 -size 10`

In the above code i am running TestDFSIO write test. I am creating 2 file of size 10.


#### Establishing Cross-Cluster trust with HPE Ezmeral Datafabric
Cross Cluster trust requests ssl_truststore data and mapr-clusters.conf data of HPE Ezmeral Datafabric. These data are passed to the pod through kubernetes secret. The data should be base64 encoded. Name of the key for ssl_truststore is truststore and name of the key for mapr-clusters.conf is conf. A sample template of this is given below
<br/><br/>
apiVersion: v1 <br/>
kind: Secret<br/>
metadata:<br/>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;name: ssl-remote<br/>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;labels:<br/>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;kubedirector.hpe.com/secretType : ssl-remote<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;data:<br/>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;truststore: <remote MapR's ssl_truststore in base64 -w0 encoding><br/>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;conf: <remote MapR's mapr-clusters.conf in base64 encoding><br/>
---<br/>
       
#### Docker image location

There are 2 images used in the app and the following are the location of images.

docker.io/bluedata/datafabric610:1.0
docker.io/bluedata/datafabric610edge:1.0