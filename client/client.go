package main

import (
	"fmt"
    "net/rpc"
)

type ClientMessgaeResponse struct {
	Response string
}

var ips = []string{
	"127.0.0.1:7070",
	"127.0.0.1:8080",
	"127.0.0.1:9090",
	"127.0.0.1:7171",
	"127.0.0.1:8181",
}

func main() {
	for _, ip := range ips {
		client, _ := rpc.DialHTTP("tcp", ip)

		var response ClientMessgaeResponse

		// TODO: Need to change this to client.Go and make it asynchronous
		_ = client.Call("State.ClientMessage",
							"request",
							&response)
		_ = client.Close()
		fmt.Println(response.Response)
	}
}