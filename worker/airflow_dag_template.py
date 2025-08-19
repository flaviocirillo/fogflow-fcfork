from datetime import datetime
from airflow import DAG
from airflow.operators.python import PythonOperator
import sys
import ngsildclient
import re


# Somethins similar to the following to import the module
# sys.path.insert(0, 'fogfunctionmodule.zip')
#
# MODULE PLACEHOLDER:

# Something similar to the following will be added by the airflowengine:
# from mypackage.somecode import handleEntity
# 
# FOGFUNCTION PLACEHOLDER:


# Something similar to the following will be added by the airflowengine:
# configurations = [{
# 	"command": "CONNECT_BROKER",
# 	"brokerURL": "http://10.11.13.150:9090" 
#     },
#     "Inputs":[{"Type":"Temperature","ID":"","AttributeList":[],"InformationModel":"NGSI-LD"}]
# ]
# 
# CONFIGURATION PLACEHOLDER:


# global variables
brokerClient = None
inputs = []
# inputs = [{"Type":"Temperature","ID":"","AttributeList":[],"InformationModel":"NGSI-LD"}]

def read_broker_ip_and_port(brokerURL, default_port="9090"):
    # Regex: optional scheme, capture IP/host, optional port
    match = re.match(r'^(?:https?://)?([^:/]+)(?::(\d+))?$', brokerURL)
    if not match:
        raise ValueError(f"Invalid broker URL: {brokerURL}")

    ip, port = match.groups()
    if port is None:  # no port provided → default
        port = default_port
    return ip, port

def readConfig():
    global brokerClient
    for config in configurations:
        if config['command'] == 'CONNECT_BROKER':
            ip, port = read_broker_ip_and_port(config['brokerURL'])
            brokerClient = ngsildclient.Client(ip, port)
        if config['command'] == 'SET_INPUTS':
            inputs.append(config)

def get_entities_by_type(entityType, geoscope=[]):
    global brokerClient
    ctxEntities = brokerClient.queryAll(f"?type={entityType}")
    return ctxEntities

def get_entity_by_id(entityId):
    global brokerClient
    ctxEntity = brokerClient.query(entityId)[0]
    return ctxEntity

def task_wrapper(**kwargs):
    readConfig()

    data = []

    for input in inputs:
        if "ID" in input and input["ID"].strip() != "":
            ctxEntities = get_entity_by_id(input["ID"])
        else:
            ctxEntities = get_entities_by_type(input["Type"])
        
        data.append({"input" : input,
                    "entities": ctxEntities})
            

    handleEntity(data)

with DAG(
    dag_id='simple_entity_handler_dag',
    schedule_interval='@once',  # Run immediately one time
    start_date=datetime(2024, 1, 1), 
    catchup=False,
    tags=['example'],
) as dag:

    run_handler = PythonOperator(
        task_id='run_handle_entity',
        python_callable=task_wrapper,
        provide_context=True,
    )
