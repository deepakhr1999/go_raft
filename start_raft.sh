#!/bin/bash

declare -a ports=(8080 7070 7171 8181 9090)

mkdir -p db

echo Starting Raft Cluster

for i in {1..5}
do
    node=${ports[$i-1]}
    ./go_raft $i :$node $1> app/${node}.txt &
    echo Started node on port $node
done

# sleep 16
# ./client/client register iit dharwad