#!/bin/bash
echo Killing Raft Cluster

declare -a ports=(8080 7070 7171 8181 9090)

for port in ${ports[*]}
do
    # fuser -k "${port}"/tcp
    ./kill_port.sh $port
done