#### KAFKA BRIEF
Kafka is a distributed stream processing, message broker service. Confluent Kafka Platform has 7 components and corresponding roles: broker, zookeeper, schema-registry, ksql, connect, rest-proxy, and control-center. An optional client pod for running test loads is also provided in the yaml file. The client pod role is test-client. The first six components are available in the community version whereas the seventh - control center has to be licensed in 30 days.
More about the Confluent Kafka cluster can be read [here.](https://docs.confluent.io/current/platform.html#cp-platform)

#### KAFKA ROLES
* Broker is the main component for Kafka loads. It is a cluster with minimum of 3 nodes. With the increase in load, number of brokers can be scaled up horizontally to be able to take more load.
* Zookeeper ensemble of 3 nodes is required for brokers. Zookeeper keeps track of broker leader and in case brokers, go down, decides on the next leader. The ensemble is of 3 nodes in most cases and can be scaled up to 5 as well. In this version we go with 3 nodes only.
* Connect helps with creation of non-default connectors to connect Kafka to other data systems such as Apache Hadoop using Kafka Connect API. 
* REST proxy provides REST API interface support for connecting to producers and consumers written in all languages. 
* KSQL is a SQL based ksqlDB is the streaming engine for Kafka. 
* Schema registry is a schema management component used by other components to adhere to a common message format.
* Control-center is s a GUI-based system used for managing and monitoring Kafka.
* Test-client is provided for testing basic functionality using sample producers and consumers. It is not a Kafka role and is optional to include during cluster creation.
 
#### STEPS FOR SAMPLE TEST RUNS ON kafka-client
* Add 2 kafka-clients in the kafka cluster.
* On one client, where you want to run the producer, run the script: 
```bash
/bin/test_client producer <topic_name>
```
* On one client, where you want to run the consumer, run the script: 
```bash
/bin/test_client consumer <topic_name>
```
* From the producer, send the messages and observe that the same would be received on the client.
 
#### QA Tests performed:
* Create a cluster. 
* Increase broker nodes, test that the messages produced and consumed without issues. Confluent Control center should show a healthy cluster.
* Reduce broker nodes, test message transfer from client and cluster state in control center.
* Power off nodes to see that the cluster health from control center is good and messages are transferred without issues.
* Run message streaming test. Sample [streaming test using KSQL.](https://kafka-tutorials.confluent.io/transform-a-stream-of-events/ksql.html) 
* Delete cluster.