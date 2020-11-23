package common

import (
   "fmt"
   "net/rpc"
   "time"
)

type Entry struct {
	Content string
	Index int
	Term int
	Commited bool
}

type State struct {
	// Persistant state on all servers
	CurrentTerm int
	VotedFor int
	Log []Entry

	// Volatile state on all servers
	CommitIndex int // Initialized to zero
	LastApplied int // Initialized to zero

	// Volatile state on leaders
	NextIndex []int
	MatchIndex []int // Initialized to zero
}

// appendEntriesResponse represents the response to an appendEntries RPC.
type appendEntriesResponse struct {
	Term    int `json:"term"`
	Success bool   `json:"success"`
	Reason  string
}

// Entry represents the request the leader asks to append
type Entry struct {
	Term int
	LeaderID int
	PrevLogIndex int
	PrevLogTerm int
	Entries []Entry
	LeaderCommit int
}

// ReqVote represents the request the condidate sends to all the other servers when it doesn't 
// receive the heartbeat Tick.
type ReqVote struct {
	Term         int `json:"term"`
	CandidateID  int `json:"candidate_id"`
	LastLogIndex int `json:"last_log_index"`
	LastLogTerm  int `json:"last_log_term"`
}

// requestVoteResponse represents the response to a requestVote RPC.
type requestVoteResponse struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"vote_granted"`
	Reason      string
}