package main

import (
	"fmt"
	// "net/rpc"
	"os"
	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
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

var options = govec.GetDefaultLogOptions()
var logger = govec.InitGoVector("client", "client", govec.GetDefaultConfig())

func main() {
	for _, ip := range ips {
		client, err := vrpc.RPCDial("tcp", ip, logger, options)
		if err!=nil{
			continue
		}

		var response ClientMessgaeResponse

		// TODO: Need to change this to client.Go and make it asynchronous
		_ = client.Call("State.ClientMessage",
							os.Args[1],
							&response)
		_ = client.Close()
		fmt.Println(response.Response)
	}
}