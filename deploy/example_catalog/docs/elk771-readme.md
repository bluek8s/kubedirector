#### ELK BRIEF

ELK" is the acronym for three open source projects: Elasticsearch, Logstash, and Kibana. Elasticsearch is a search and analytics engine. Logstash is a server‑side data processing pipeline that ingests data from multiple sources simultaneously, transforms it, and then sends it to a "stash" like Elasticsearch. Kibana lets users visualize data with charts and graphs in Elasticsearch.
ELK has following roles:

* master(mandatory)
* data(optional)
* kibana(optional)
* logstash(optional)

More about the ELK cluster can be read [here.](https://www.elastic.co/what-is/elk-stack)

#### ELK ROLES

* master and data roles are running elasticsearch service.
* kibana role is running kibana service
* logstash role is running logstash service

#### STEPS FOR SAMPLE TEST RUNS 

 Login to Kibana UI and navigate to Kibana's console tool. Following queries can be run from Kibana:

 1. Create product indice

    Send request

        PUT /products
        {
          "settings": 
          {
           "number_of_replicas": 2,
              "number_of_shards": 2
          }
        }

     Verify that response status code is 200 Ok and the response is 

        {
          "acknowledged" : true,
          "shards_acknowledged" : true,
          "index" : "products"
        }

2. Create document in product indice

     Send request
   
       POST /products/_doc/100
       {
          "name": "product1",
          "price": 64,
          "in_stock": 10
       }

      Verify that response code is 201 Created, and the document got created with id as 100 by verifying the response

         {
          "_index" : "products",
          "_type" : "_doc",
          "_id" : "100",
          "_version" : 1,
          "result" : "created",
          "_shards" : {
            "total" : 3,
            "successful" : 2,
            "failed" : 0
          },
          "_seq_no" : 1,
          "_primary_term" : 1
        }

3. Retrieve the document created with id as 100 and verify the response

   Send request

   GET /products/_doc/100

   Response

       {
        "_index" : "products",
        "_type" : "_doc",
        "_id" : "100",
        "_version" : 1,
        "_seq_no" : 1,
        "_primary_term" : 1,
        "found" : true,
        "_source" : {
          "name" : "product1",
          "price" : 64,
          "in_stock" : 10
        }
       }

   Similarly we can perform update, delete on the product indice by sending rest request from the console tool in Kibana.

#### Docker Image location

* docker.io/bluedata/elk771

#### Docker Pull command

* docker pull docker.io/bluedata/elk771:1.1