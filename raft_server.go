package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"

	"net"
	"net/rpc"
	"os"
	"time"

	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
)

const (
	follower  = "Follower"
	candidate = "Candidate"
	leader    = "Leader"
)

const (
	noVote = "0"
)

var ips = []string{
	"127.0.0.1:7070",
	"127.0.0.1:8080",
	"127.0.0.1:9090",
	"127.0.0.1:7171",
	"127.0.0.1:8181",
}
var writeTicker = *time.NewTicker(25 * time.Second)
var options = govec.GetDefaultLogOptions()
var logger = govec.InitGoVector("server"+os.Args[2], "logs/node"+os.Args[2], govec.GetDefaultConfig())

var dummyArgs = SafeDummyType{
	Data: "dummy",
}
var dummyResp = SafeDummyType{
	Data: "dummy",
}

// WriteFile write the logs to the file
func (ServerState *State) WriteFile(args *SafeDummyType, response *SafeDummyType) error {
	for range writeTicker.C {
		writable := PersistentStorage{
			VotedFor:    ServerState.VotedFor,
			Log:         ServerState.Log,
			CommitIndex: ServerState.CommitIndex,
			LastApplied: ServerState.LastApplied,
		}

		b, _ := json.MarshalIndent(writable, "", " ")
		filename := fmt.Sprintf("db/Node%s.json", ServerState.CandidateID)
		err := ioutil.WriteFile(filename, b, 0644)
		if err != nil {
			fmt.Printf("Node%s: error writing file\n", ServerState.CandidateID)
		}
	}
	return nil
}

// ResetHeartbeat creates a new timer for hearbeat
func (ServerState *State) ResetHeartbeat(args *SafeDummyType, response *SafeDummyType) error {
	ServerState.Heartbeat = *time.NewTicker(6 * time.Second)
	return nil
}

// ResetElectionTimeout creates new timer for election timeout
func (ServerState *State) ResetElectionTimeout(args *SafeDummyType, response *SafeDummyType) error {
	// anarchy problem is solved using randomness
	// sampling from the millisecond realm allows greater variation
	randTimeoutInMilliSeconds := rand.Intn(6000-200) + 200
	ServerState.ElectionTimeout = *time.NewTicker(time.Duration(randTimeoutInMilliSeconds) * time.Millisecond)
	return nil
}

// CheckHeartbeat prevents a follower from becoming a candidate
func (ServerState *State) CheckHeartbeat(args *SafeDummyType, response *SafeDummyType) error {
	for range ServerState.Heartbeat.C {
		if ServerState.State == follower {
			// fmt.Println("I AM FOLLOWER", ServerState.CandidateID, ServerState.CurrentTerm, ServerState.CommitIndex)
			ServerState.State = candidate
			ServerState.ResetElectionTimeout(&dummyArgs, &dummyResp)
		}
	}
	return nil
}

// CheckElectionTimeout counts votes and allows a candidate to become a leader
func (ServerState *State) CheckElectionTimeout(args *SafeDummyType, response *SafeDummyType) error {
	time.Sleep(15 * time.Second)
	for range ServerState.ElectionTimeout.C {
		if ServerState.State == candidate {
			fmt.Printf("Node %s: Candidate, term: %d, comIdx: %d\n", ServerState.CandidateID, ServerState.CurrentTerm, ServerState.CommitIndex)
			ServerState.CurrentTerm = ServerState.CurrentTerm + 1
			ServerState.VotedFor = ServerState.CandidateID
			ServerState.ResetElectionTimeout(&dummyArgs, &dummyResp)

			// Instance of ReqVote
			term := 0
			if len(ServerState.Log) == 0 {
				term = 0
			} else {
				term = ServerState.Log[len(ServerState.Log)-1].Term
			}
			request := ReqVote{
				Term:         ServerState.CurrentTerm,
				CandidateID:  ServerState.CandidateID,
				LastLogIndex: ServerState.CommitIndex,
				LastLogTerm:  term,
			}

			// request other nodes to vote and gather their votes
			totVotes := 0
			for _, ip := range ips {
				client, err := vrpc.RPCDial("tcp", ip, logger, options)
				if err != nil {
					continue
				}

				var response RequestVoteResponse

				_ = client.Call("State.HandleRequestVote", request, &response)

				if response.VoteGranted {
					totVotes++
				}
				_ = client.Close()
			}

			// if we have sufficient votes become a leader
			if totVotes >= 3 {
				ServerState.State = leader
				fmt.Printf("Node %s: Leader, term: %d, comIdx: %d\n", ServerState.CandidateID, ServerState.CurrentTerm, ServerState.CommitIndex)

				// Change NextIndex and MatchIndex
				ServerState.MatchIndex = []int{0, 0, 0, 0, 0}
				for i := range ServerState.NextIndex {
					ServerState.NextIndex[i] = len(ServerState.Log) + 1
				}
			} else {
				ServerState.ResetElectionTimeout(&dummyArgs, &dummyResp)
			}
		}
	}
	return nil
}

// SendHeartbeat is called by leader
func (ServerState *State) SendHeartbeat(args *SafeDummyType, response *SafeDummyType) error {
	time.Sleep(15 * time.Second)
	for {
		if ServerState.State == leader {
			// Heartbeat with no entries
			var entries []Entry
			// Instance of AppEntry
			term := 0
			if len(ServerState.Log) != 0 {
				term = ServerState.Log[len(ServerState.Log)-1].Term
			}

			// implement for loop to send request
			for _, ip := range ips {
				client, err := vrpc.RPCDial("tcp", ip, logger, options)
				if err != nil {
					continue
				}

				request := AppEntry{
					Term:         ServerState.CurrentTerm,
					LeaderID:     ServerState.CandidateID,
					PrevLogIndex: len(ServerState.Log) - 1,
					PrevLogTerm:  term,
					Entries:      entries,
					LeaderCommit: ServerState.CommitIndex,
				}

				var response AppendEntriesResponse

				// TODO: Need to change this to client.Go and make it asynchronous
				_ = client.Call("State.HandleAppendEntries",
					request,
					&response)
				_ = client.Close()
			}
		}
	}
}

// HandleRequestVote function
func (ServerState *State) HandleRequestVote(args *ReqVote, response *RequestVoteResponse) error {

	// If the request is from an old term, reject
	if args.Term < ServerState.CurrentTerm {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = fmt.Sprintf("Term %d > %d;", ServerState.CurrentTerm, args.Term)
		return nil
	}

	// If the request is from a newer term, reset our state
	stepDown := false
	if args.Term > ServerState.CurrentTerm {
		ServerState.CurrentTerm = args.Term
		ServerState.VotedFor = noVote
		ServerState.State = follower
		ServerState.ResetElectionTimeout(&dummyArgs, &dummyResp)
		stepDown = true
	}

	// Special case: if we're the leader, and we haven't been deposed by a more
	// recent term, then we should always deny the vote
	if ServerState.State == leader && !stepDown {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = "Already the leader"
		return nil
	}

	// Reject if already voted for someone else
	if ServerState.VotedFor != noVote && ServerState.VotedFor != args.CandidateID {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = fmt.Sprintf("Already Cast vote for %s", ServerState.VotedFor)
		return nil
	}

	// If you send yourseld the request,

	// If candidate log isn't up-to-date, don't vote
	if ServerState.CommitIndex > args.LastLogIndex {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = "Log is not up to date"
		return nil
	}

	// If all good till now, VOTE!!
	ServerState.VotedFor = args.CandidateID
	fmt.Printf("Node %s: Voted node%s, term: %d, comIdx:%d\n", ServerState.CandidateID, args.CandidateID, ServerState.CurrentTerm, ServerState.CommitIndex)
	ServerState.ResetElectionTimeout(&dummyArgs, &dummyResp)
	ServerState.ResetHeartbeat(&dummyArgs, &dummyResp)
	response.Term = ServerState.CurrentTerm
	response.VoteGranted = true
	response.Reason = ""
	return nil
}

// HandleAppendEntries is called by everyone when leader sends request
func (ServerState *State) HandleAppendEntries(args *AppEntry, response *AppendEntriesResponse) error {
	// if I leader and my heart beat then reject haha
	if ServerState.CandidateID == args.LeaderID && len(args.Entries) == 0 {
		response.Term = ServerState.CurrentTerm
		response.Success = false
		response.Reason = fmt.Sprintf("I reject my heart beat")
		return nil
	}
	// If the request is from an old term, reject
	if ServerState.CurrentTerm > args.Term {
		response.Term = ServerState.CurrentTerm
		response.Success = false
		response.Reason = fmt.Sprintf("Term %d > %d", ServerState.CurrentTerm, args.Term)
		return nil
	}

	// stepDown means you need to: s.leader=r.LeaderID, s.state.Set(Follower).
	stepDown := false

	// If the request is from a newer term, reset our state
	if ServerState.CurrentTerm < args.Term {
		// fmt.Println(ServerState.CurrentTerm, args.Term)
		ServerState.State = follower
		ServerState.CurrentTerm = args.Term
		ServerState.VotedFor = noVote
		stepDown = true
	}

	// Special case for candidates: "While waiting for votes, a candidate may
	// receive an appendEntries RPC from another server claiming to be leader.
	// If the leader’s term (included in its RPC) is at least as large as the
	// candidate’s current term, then the candidate recognizes the leader as
	// legitimate and steps down, meaning that it returns to follower state."
	if ServerState.CurrentTerm == args.Term && ServerState.State == candidate {
		// fmt.Println(ServerState.CurrentTerm, args.Term, ServerState.State)
		ServerState.State = follower
		ServerState.CurrentTerm = args.Term
		ServerState.VotedFor = noVote
		stepDown = true
	}

	// ========================= DEBUG ======================
	// fmt.Println("Recieved heartbeat from", args.LeaderID, ServerState.CurrentTerm, ServerState.CommitIndex)

	ServerState.ResetHeartbeat(&dummyArgs, &dummyResp)

	if stepDown == true {
		ServerState.State = follower
	}

	// Reject if log doesn't contain a matching previous entry
	pos := 0
	for ; pos < len(ServerState.Log); pos++ {
		if ServerState.Log[pos].Index < args.PrevLogIndex {
			continue
		}
		if ServerState.Log[pos].Index == args.PrevLogIndex {
			if ServerState.Log[pos].Term != args.PrevLogTerm {
				response.Term = ServerState.CurrentTerm
				response.Success = false
				response.Reason = "Inconsistant"
				return nil
			}
		}
	}

	// Truncate from current position
	if pos != 0 && pos+1 < len(ServerState.Log) {
		ServerState.Log = ServerState.Log[:pos+1]
	}

	// Append logs
	// appendingLogs := args.Entries[pos + 1:]
	ServerState.Log = append(ServerState.Log, args.Entries...)

	// If all good till now change state of the Server
	ServerState.CommitIndex = args.LeaderCommit
	ServerState.LastApplied = len(ServerState.Log)
	response.Term = ServerState.CurrentTerm
	response.Success = true
	response.Reason = "SUCCESS"
	return nil
}

func main() {
	/*
		There is a case where when a leader is dead,
		all followers rise to candidates and vote for themselves.
		This leads to anarchy.
	*/
	rand.Seed(time.Now().UTC().UnixNano()) // fix for anarchy is randomness

	// Include all variables for server state
	var MyLog []Entry
	nextIndex := []int{1, 1, 1, 1, 1}
	matchIndex := []int{0, 0, 0, 0, 0}
	ticker := *time.NewTicker(6 * time.Second)
	ticker1 := *time.NewTicker(2 * time.Second)

	// Make new instance of State
	ServerState := &State{
		CurrentTerm: 0,
		VotedFor:    "0",
		Log:         MyLog,
		State:       follower,
		CandidateID: os.Args[1],

		// Volatile state on all servers
		CommitIndex: 0,
		LastApplied: 0,
		Heartbeat:   ticker,

		// Volatile among candidates
		ElectionTimeout: ticker1,

		// Volatile state on leaders
		NextIndex:  nextIndex,
		MatchIndex: matchIndex}

	filename := fmt.Sprintf("db/Node%s.json", os.Args[1])
	if _, err := os.Stat(filename); err == nil {
		//change the initialization if the file exists
		storageAsString, _ := ioutil.ReadFile(filename)

		storage := new(PersistentStorage)
		json.Unmarshal(storageAsString, &storage)
		ServerState.VotedFor = storage.VotedFor
		ServerState.Log = storage.Log
		ServerState.CommitIndex = storage.CommitIndex
		ServerState.LastApplied = storage.LastApplied
	}
	server := rpc.NewServer()

	// rpc.Register(ServerState)
	server.Register(ServerState)

	// rpc.HandleHTTP()
	l, e := net.Listen("tcp", os.Args[2])
	if e != nil {
		log.Fatal("listen error:", e)
	}

	// Does everything a follower has to do
	go ServerState.CheckHeartbeat(&dummyArgs, &dummyResp)

	// Does everything a candidate has to do
	go ServerState.CheckElectionTimeout(&dummyArgs, &dummyResp)

	// Does everything a leader has to do
	go ServerState.SendHeartbeat(&dummyArgs, &dummyResp)

	go ServerState.WriteFile(&dummyArgs, &dummyResp)

	vrpc.ServeRPCConn(server, l, logger, options)

	/*
		ServerState is FOLLOWER

		TODO:
		if heart beat interval times out:
		switch to CANDIDATE
		wait for timeout
		else:
		do nothing

		ServerState is CANDIDATE

		TODO:
		if not timeout:
					Gather votes
					if majority votes:
						switch LEADER
				else:
					Increment currentTerm
					Vote for self
					Reset election timer
					Send RequestVote

		ServerState is LEADER

	*/

}
