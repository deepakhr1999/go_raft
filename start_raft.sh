#!/bin/bash

declare -a ports=(8080 7070 7171 8181 9090)

echo Starting Raft Cluster

for i in {1..5}
do
    node=${ports[$i-1]}
    ./go_raft $i :$node > out/${node}.txt &
    echo Started node on port $node
done