#!/bin/bash

echo -e "Register Operator"
curl 'http://localhost:8080/operator' -X POST -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:130.0) Gecko/20100101 Firefox/130.0' -H 'Accept: application/json' -H 'Accept-Language: en-US,en;q=0.5' -H 'Accept-Encoding: gzip, deflate, br, zstd' -H 'Referer: http://localhost:8080/operator.html' -H 'Content-Type: application/json' -H 'Origin: http://localhost:8080' -H 'DNT: 1' -H 'Connection: keep-alive' -H 'Sec-Fetch-Dest: empty' -H 'Sec-Fetch-Mode: cors' -H 'Sec-Fetch-Site: same-origin' -H 'Priority: u=0' --data-raw '[{"name":"dummyf","description":"","parameters":[],"dockerimages":[],"designboard":{"edges":[],"blocks":[{"id":1,"x":-282.9999999999999,"y":-114.00000000000009,"type":"Operator","module":null,"values":{"name":"dummyf","description":""}}]}}]'

sleep 5;


echo -e "\nRegister Container"
curl 'http://localhost:8080/dockerimage/dummyf' -X POST -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:130.0) Gecko/20100101 Firefox/130.0' -H 'Accept: application/json' -H 'Accept-Language: en-US,en;q=0.5' -H 'Accept-Encoding: gzip, deflate, br, zstd' -H 'Referer: http://localhost:8080/operator.html' -H 'Content-Type: application/json' -H 'Origin: http://localhost:8080' -H 'DNT: 1' -H 'Connection: keep-alive' -H 'Sec-Fetch-Dest: empty' -H 'Sec-Fetch-Mode: cors' -H 'Sec-Fetch-Site: same-origin' -H 'Priority: u=0' --data-raw '{"name":"fogflow/dummy","hwType":"X86","osType":"Linux","operatorName":"dummyf","prefetched":false,"tag":"latest"}'

sleep 5;

echo -e "\nRegister FogFunction"
curl 'http://localhost:8080/fogfunction' -X POST -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:130.0) Gecko/20100101 Firefox/130.0' -H 'Accept: application/json' -H 'Accept-Language: en-US,en;q=0.5' -H 'Accept-Encoding: gzip, deflate, br, zstd' -H 'Referer: http://localhost:8080/function.html' -H 'Content-Type: application/json' -H 'Origin: http://localhost:8080' -H 'DNT: 1' -H 'Connection: keep-alive' -H 'Sec-Fetch-Dest: empty' -H 'Sec-Fetch-Mode: cors' -H 'Sec-Fetch-Site: same-origin' -H 'Priority: u=0' --data-raw '[{"name":"testbytype","topology":{"name":"testbytype","description":"","tasks":[{"name":"testbytype","operator":"dummyf","input_streams":[{"selected_type":"Floor","selected_attributes":[],"groupby":"EntityType","scoped":false}],"output_streams":[{"entity_type":"Out"}]}]},"intent":{"id":"ServiceIntent.94a14f13-6aea-4fdf-a04c-a1824699b799","topology":"testbytype","priority":{"exclusive":false,"level":0},"qos":"default","geoscope":{"scopeType":"global","scopeValue":"global"}},"designboard":{"edges":[{"id":1,"block1":2,"connector1":["stream","output"],"block2":1,"connector2":["streams","input"]}],"blocks":[{"id":1,"x":5,"y":-156,"type":"Task","module":null,"values":{"name":"testbytype","operator":"dummyf","outputs":["Out"]}},{"id":2,"x":-301,"y":-164,"type":"EntityStream","module":null,"values":{"selectedtype":"Floor","selectedattributes":["all"],"groupby":"EntityType","scoped":false}}]},"status":"enabled"}]'

sleep 5;

echo -e "\nSend data"
curl --location 'http://localhost:8070/ngsi-ld/v1/entities' \
--header 'Content-Type: application/ld+json' \
--header 'Fiware-Correlator: Task.testbytype.testbytype.3083993709' \
--data-raw '{
  "id": "house2:smartrooms:Floor:floor006",
  "type": "Floor",
  "temperature": {
        "value": 24,
        "unitCode": "CEL",
        "type": "Property",
        "providedBy": {
                "type": "Relationship",
                "object": "smartbuilding:house2:sensor0815"
         }
   },
  "isPartOf": {
        "type": "Relationship",
        "object": "smartcity:houses:house2"
  },
  "@context": [{"Room": "urn:mytypes:room", "temperature": "myuniqueuri:temperature", "isPartOf": "myuniqueuri:isPartOf"},"https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"]
}'
