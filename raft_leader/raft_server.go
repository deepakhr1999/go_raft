package main

import (
   "fmt"
   "net/rpc"
   "net/http"
   "time"
   "os"
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

func (ServerState *State) ResetHeartbeat() {
    ServerState.Heartbeat = *time.NewTicker(3 * time.Second)
}

func (ServerState *State) ResetElectionTimeout() {
    ServerState.ElectionTimeout = *time.NewTicker(2 * time.Second)
}

func (ServerState *State) CheckHeartbeat() {
	for _ = range ServerState.Heartbeat.C {
		if ServerState.State == follower {
			fmt.Println("I AM FOLLOWER", ServerState.CandidateID, ServerState.CurrentTerm)
			ServerState.State = candidate
			ServerState.ResetElectionTimeout()
		}
	}
    
}

func (ServerState *State) CheckElectionTimeout() {
	time.Sleep(15 * time.Second)
	for _ = range ServerState.ElectionTimeout.C {
		if ServerState.State == candidate {
			fmt.Println("I AM CANDIDATE", ServerState.CandidateID, ServerState.CurrentTerm)
			ServerState.CurrentTerm = ServerState.CurrentTerm + 1
			ServerState.VotedFor = ServerState.CandidateID
			ServerState.ResetElectionTimeout()

			// Instance of ReqVote
			request := ReqVote{
							Term:ServerState.CurrentTerm,
							CandidateID:ServerState.CandidateID,
							LastLogIndex:0,
							LastLogTerm:0,
						}

			// implement for loop to send request
			totVotes := 0
			for _, ip := range ips {
				client, err := rpc.DialHTTP("tcp", ip)
				if err != nil {
					continue
				}

				var response RequestVoteResponse

				// TODO: Need to change this to client.Go() and make it asynchronous
				_ = client.Call("State.HandleRequestVote",
								 request,
								 &response)
				if response.VoteGranted {
					totVotes++
				}
				_ = client.Close()
			}
			if totVotes >= 3 {
				ServerState.State = leader
			}
		}
	}
}

func (ServerState *State) GatherCheck() {
	for{
		if ServerState.State == candidate {
			// Gather votes and become leader if sufficient
		}
	}
}

func (ServerState *State) SendHeartbeat() {
	time.Sleep(15 * time.Second)
	for{
		if ServerState.State == leader {
			fmt.Println("I AM LEADER", ServerState.CandidateID, ServerState.CurrentTerm)
			// Heartbeat with no entries
			var entries []Entry
			// Instance of AppEntry
			request := AppEntry{
							Term:ServerState.CurrentTerm,
							LeaderID:ServerState.CandidateID,
							PrevLogIndex:0,
							PrevLogTerm:0,
							Entries:entries,
							LeaderCommit:0,
						}

			// implement for loop to send request
			for _, ip := range ips {
				client, _ := rpc.DialHTTP("tcp", ip)
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
		response.Reason = fmt.Sprintf("Already Cast vote for %d", ServerState.VotedFor)
		return nil
	}

	// If candidate log isn't up-to-date, don't vote
	if ServerState.CommitIndex > args.LastLogIndex {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = "Log is not up to date"
		return nil
	}

	// If all good till now, VOTE!!
	ServerState.VotedFor = args.CandidateID
	response.Term = ServerState.CurrentTerm
	response.VoteGranted = true
	response.Reason = ""
	return nil
}



// stepDown means you need to: s.leader=r.LeaderID, s.state.Set(Follower).
func (ServerState *State) HandleAppendEntries(args *AppEntry, response *AppendEntriesResponse) error {
	
	// If the request is from an old term, reject
	if ServerState.CurrentTerm > args.Term {
		response.Term = ServerState.CurrentTerm
		response.Success = false
		response.Reason = fmt.Sprintf("Term %d > %d", ServerState.CurrentTerm, args.Term)
		return nil
	}
	
	// If the request is from a newer term, reset our state
	stepDown := false
	if ServerState.CurrentTerm < args.Term {
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
		ServerState.CurrentTerm = args.Term
		ServerState.VotedFor = noVote
		stepDown = true
	}

	// ========================= DEBUG ======================
	fmt.Println(stepDown, ServerState.CurrentTerm)
	
	// TODO: Implement timer election
	ServerState.ResetHeartbeat()

	// Reject if log doesn't contain a matching previous entry
	pos := 0
	for ; pos < len(ServerState.Log); pos++ {
		if ServerState.Log[pos].Index < args.PrevLogIndex {
			continue
		}
		if ServerState.Log[pos].Index == args.PrevLogIndex {
			if ServerState.Log[pos].Term != args.PrevLogTerm{
				response.Term = ServerState.CurrentTerm
				response.Success = false
				response.Reason = "Inconsistant"
				return nil
			}
		}
	}

	// Truncate from current position
	if pos != 0 {
		ServerState.Log = ServerState.Log[:pos + 1]
	}

	// Append logs
	ServerState.Log = append(ServerState.Log, args.Entries...)

	// If all good till now change state of the Server
	ServerState.CommitIndex = args.LeaderCommit
	ServerState.LastApplied = pos + len(args.Entries)
	response.Term = ServerState.CurrentTerm
	response.Success = true
	response.Reason = ""
	return nil
}

func main() {
	
	// Include all variables for server state
	// TODO: Improve naming convention
	var MyLog []Entry
	nextIndex := []int{1, 1, 1, 1, 1}
	matchIndex := []int{0, 0, 0, 0, 0}
	ticker := *time.NewTicker(3 * time.Second)
	ticker1 := *time.NewTicker(2 * time.Second)


	// Make new instance of State
	ServerState := &State{
		CurrentTerm:0,
		VotedFor:"0",
		Log:MyLog,
		State:leader,
		CandidateID:os.Args[1],

		// Volatile state on all servers
		CommitIndex:0,
		LastApplied:0,
		Heartbeat:ticker,

		// Volatile among candidates
		ElectionTimeout:ticker1,


		// Volatile state on leaders
		NextIndex:nextIndex,
		MatchIndex:matchIndex}

	rpc.Register(ServerState)

	rpc.HandleHTTP()

	// Does everything a follower has to do
	go ServerState.CheckHeartbeat()

	// Does everything a candidate has to do
	go ServerState.CheckElectionTimeout()
	
	// Does everything a leader has to do
	go ServerState.SendHeartbeat()

	err := http.ListenAndServe(os.Args[2], nil)
	if err != nil {
		fmt.Println(err.Error())
	}


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

	TODO:
			send heart beat
			print("I AM THE FUCKING LEADER")

	*/

}