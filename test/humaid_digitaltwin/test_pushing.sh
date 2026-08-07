#!/bin/bash

for i in {1..400}
do
    echo -n -e "$i"'\r'
       
   curl --location 'http://localhost:8070/ngsi-ld/v1/entities' \
  --header 'Content-Type: application/ld+json' \
  --header 'Fiware-Correlator: Task.testbytype.testbytype.3083993709' \
  --data-raw '{
    "id": "house2:smartrooms:Floor:floor'"$i"'",
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
done

echo ""
