## Hi!

This project consists of two microservices that store and show calls history the user inside his company.

Calls can be of the following types:
 - incoming (from client to employee)
 - outgoing (from employee to client)

## Microservices
### call-collector
#### What does service do?
 - collect JSON call details by http
 - parse details to resulting JSON
 - produce result to Kafka topic (using Kafka compatible Redpanda)

### ClickHouse DB
#### Tables
 - _calls_ - table for storing calls
 - _calls_queue_ - table as Kafka consumer
 - _calls_mv_ - materialized view as proxy between _calls_queue_ and _calls_

TODO: move JSON parser from call-collector to materialized view

### rest-server
Service realize REST API for user to get calls history or calls statistics

## Getting Started!

```
> docker-compose up -d

POST some call:
> curl -X POST --location "http://localhost:50010/call" \
    -H "Content-Type: application/json" \
    -d "{
          \"departmentID\": \"company1\",
          \"departmentName\": \"Company 1\",
          \"start\": \"2022-05-17T15:42:21.795Z\",
          \"connect\": \"2022-05-17T15:42:25.795Z\",
          \"disconnect\": \"2022-05-17T15:44:24.795Z\",
          \"callType\": \"outgoing\",
          \"employee\": {
            \"number\": 79990001122,
            \"name\": \"Ivan\"
          },
          \"client\": 79201234566,
          \"uuid\": \"6937d911-5e36-4849-8415-c55c21fe2eea\",
          \"additionalInfo\": []
        }"

GET call:
> curl -X GET --location "http://localhost:50020/history/company1/?start=20220517T000000Z&end=20220518T000000Z" \
    -H "Content-Type: application/json"
```