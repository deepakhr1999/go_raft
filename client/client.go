package main

import (
	"fmt"
    "net/rpc"
)

type ClientMessgaeResponse struct {
	Response string
}

func main() {
	client, _ := rpc.DialHTTP("tcp", "127.0.0.1:9090")

	var response ClientMessgaeResponse

	// TODO: Need to change this to client.Go and make it asynchronous
	_ = client.Call("State.ClientMessage",
						"request",
						&response)
	_ = client.Close()
	fmt.Println(response.Response)
}