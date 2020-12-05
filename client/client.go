package main

import (
	"fmt"
	"os"

	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
)

// ClientMessage has the request args
type ClientMessage struct {
	Method string
	Name   string
	Pass   string
}

// ClientMessgaeResponse has the response string
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
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./client <register|login> <user> <password>")
		return
	}

	if os.Args[1] != "register" && os.Args[1] != "login" {
		fmt.Println("Usage: ./client <register|login> <user> <password>")
		return
	}

	args := ClientMessage{
		Method: os.Args[1],
		Name:   os.Args[2],
		Pass:   os.Args[3],
	}

	connRefused := true
	for _, ip := range ips {
		client, err := vrpc.RPCDial("tcp", ip, logger, options)
		if err != nil {
			continue
		}

		var response ClientMessgaeResponse

		// TODO: Need to change this to client.Go and make it asynchronous
		client.Call("State.ClientMessage", args, &response)
		client.Close()

		if response.Response != "NOT LEADER" {
			connRefused = false
			fmt.Println(response.Response)
		}
	}

	if connRefused {
		fmt.Println("Couldn't connect to any nodes on raft")
	}
}
