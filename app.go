package main

/*
	This file contains app specific implementation
*/

import (
	"fmt"

	"github.com/DistributedClocks/GoVector/govec/vrpc"
)

// ClientMessage is called by the client
func (ServerState *State) ClientMessage(args *ClientMessage, response *ClientMessgaeResponse) error {

	// do not act if you are not a leader
	if ServerState.State != leader {
		response.Response = "NOT LEADER"
		return nil
	}

	//show that the method was called
	fmt.Printf("Client called %s method, args=%s,%s \n", args.Method, args.Name, args.Pass)
	// if method is login, then iterate over all logs and check for string
	if args.Method == "login" {
		target := fmt.Sprintf("%s %s", args.Name, args.Pass)

		response.Response = "Not Authenticated"
		for idx, entry := range ServerState.Log {
			// Not found if nothing is till commit Index
			if idx > ServerState.CommitIndex {
				break
			}

			// case target is found
			if entry.Content == target {
				response.Response = "Authenticated"
			}
		}
		return nil
	}

	// if method is register, then add entry with content = 'name pass' to the database
	entry := Entry{
		// Content:  *args,
		Content:  fmt.Sprintf("%s %s", args.Name, args.Pass),
		Index:    len(ServerState.Log),
		Term:     ServerState.CurrentTerm,
		Commited: false}
	EntryArr := []Entry{}
	EntryArr = append(EntryArr, entry)

	term := 0
	if len(ServerState.Log) != 0 {
		term = ServerState.Log[len(ServerState.Log)-1].Term
	}

	request := AppEntry{
		Term:         ServerState.CurrentTerm,
		LeaderID:     ServerState.CandidateID,
		PrevLogIndex: len(ServerState.Log) - 1,
		PrevLogTerm:  term,
		Entries:      EntryArr,
		LeaderCommit: ServerState.CommitIndex,
	}

	// implement for loop to send request
	ans := 0
	for _, ip := range ips {
		client, err := vrpc.RPCDial("tcp", ip, logger, options)
		if err != nil {
			continue
		}

		var response1 AppendEntriesResponse

		_ = client.Call("State.HandleAppendEntries", request, &response1)
		_ = client.Close()
		if response1.Success == true {
			ans++
		}
	}
	if ans > 2 {
		// response.Response = "TRUE " + strconv.Itoa(len(ServerState.Log))
		// fmt.Println("LEADER")
		response.Response = fmt.Sprintf("Register success: user %s was added to the database", args.Name)
		ServerState.CommitIndex++
	} else {
		response.Response = fmt.Sprintf("Register failed: user %s not committed, lack of votes", args.Name)
	}
	return nil
}
