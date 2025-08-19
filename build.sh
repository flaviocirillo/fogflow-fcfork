#!/bin/bash

if (( $# !=  1 )); then
	echo "Illegal number of parameters"
	echo "usage: ./build [multistage|development|arm]"
	echo "For "development" or  "arm" options to work, golang must be setup in the system." 
	exit 1
fi

cd discovery
echo "======start to build fogflow/discovery======"
./build $1 || { echo "Build failed for fogflow/discovery"; exit 1; }

cd ../broker
echo "======start to build fogflow/broker======"
./build $1 || { echo "Build failed for fogflow/broker"; exit 1; }

cd ../master
echo "======start to build fogflow/master======"
./build $1 || { echo "Build failed for fogflow/master"; exit 1; }

cd ../worker
echo "======start to build fogflow/worker======"
./build $1 || { echo "Build failed for fogflow/worker"; exit 1; }

cd ../designer
echo "======start to build fogflow/designer======"
./build $1 || { echo "Build failed for fogflow/designer"; exit 1; }

cd ../
echo "!!!finished building all FogFlow core components and generating their docker images!!!"
