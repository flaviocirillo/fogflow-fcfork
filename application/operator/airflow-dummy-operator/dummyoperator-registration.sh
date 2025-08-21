curl --location 'http://localhost:8080/operator' \
--header 'Content-Type: application/json' \
--data '[{
        "name": "testPythonOperator",
        "description": "FirstPythonOperator",
        "parameters": [],
        "pythonpackage" : 
            {
                "moduleName": "dummymodule", 
                "moduleVersion": "1.0.0", 
                "package": "dummy.dummy",  
                "prefetched": false
            }
    }]'