package common

import (
   "fmt"
   "net/rpc"
   "time"
   "common"
)

const (
	follower  = "Follower"
	candidate = "Candidate"
	leader    = "Leader"
)

const (
	ServerState := common.State{
					CurrentTerm:0
					VotedFor:0
					Log:[]
					State:candidate

					// Volatile state on all servers
					CommitIndex:0
					LastApplied:0

					// Volatile state on leaders
					NextIndex:[1, 1, 1, 1, 1]
					MatchIndex:[0, 0, 0, 0, 0]
				}
	noVote = 0
)

type Listener int



func (l *Listener) HandleAppendEntries(args *common.ReqVote, response *common.requestVoteResponse) error {

	// If the request is from an old term, reject
	if args.Term < ServerState.CurrentTerm {
		response.Term = ServerState.CurrentTerm
		response.VoteGranted = false
		response.Reason = fmt.Sprintf("Term %d > %d;", ServerState.currentTerm, args.term)
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
	if ServerState.VotedFor != 0 && ServerState.VotedFor != args.CandidateID {
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
	response.VoteGranted = false
	response.Reason = ""
	return nil
}



// stepDown means you need to: s.leader=r.LeaderID, s.state.Set(Follower).
func (l *Listener) HandleAppendEntries(args *common.Entry, response *common.appendEntriesResponse) error {
	
	// If the request is from an old term, reject
	if ServerState.currentTerm > args.Term {
		response.Term = ServerState.currentTerm
		response.Success = false
		response.Reason = fmt.Sprintf("Term %d > %d", ServerState.currentTerm, args.term)
		return nil
	}
	
	// If the request is from a newer term, reset our state
	stepDown := false
	if ServerState.currentTerm < args.Term {
		ServerState.currentTerm = args.Term
		ServerState.VotedFor = noVote
		stepDown = true
	}
	
	// Special case for candidates: "While waiting for votes, a candidate may
	// receive an appendEntries RPC from another server claiming to be leader.
	// If the leader’s term (included in its RPC) is at least as large as the
	// candidate’s current term, then the candidate recognizes the leader as
	// legitimate and steps down, meaning that it returns to follower state."
	if ServerState.currentTerm == args.Term && ServerState.State == candidate {
		ServerState.currentTerm = args.Term
		ServerState.VotedFor = noVote
		stepDown = true
	}

	// TODO: Implement timer election

	// Reject if log doesn't contain a matching previous entry
	pos := 0
	for ; pos < len(ServerState.Log); pos++ {
		if ServerState.Log[pos].Index < args.PrevLogIndex {
			continue
		}
		if ServerState.Log[pos].Index == args.PrevLogIndex {
			if ServerState.Log[pos].Term != args.PrevLogTerm{
				response.Term = ServerState.currentTerm
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
	append(ServerState.Log, args.Entries...)

	// If all good till now change state of the Server
	ServerState.CommitIndex = args.LeaderCommit
	ServerState.LastApplied = pos + len(args.Entries)
	response.Term = ServerState.currentTerm
	response.Success = true
	response.Reason = ""
	return nil
}