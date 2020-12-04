package main

import (
	"time"
)

// ClientMessage will have Method name and pass
type ClientMessage struct {
	Method string
	Name   string
	Pass   string
}

// Entry represents Log entry
type Entry struct {
	Content  string
	Index    int
	Term     int
	Commited bool
}

// State persists on all servers
type State struct {
	CurrentTerm int
	VotedFor    string
	Log         []Entry
	State       string
	CandidateID string

	// Volatile state on all servers
	CommitIndex int // Initialized to zero
	LastApplied int // Initialized to zero
	Heartbeat   time.Ticker

	// Volatile among candidates
	ElectionTimeout time.Ticker

	// Volatile state on leaders
	NextIndex  []int
	MatchIndex []int // Initialized to zero
}

// AppendEntriesResponse represents the response to an appendEntries RPC.
type AppendEntriesResponse struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
	Reason  string
}

// AppEntry contains Entry, represents the request the leader asks to append
type AppEntry struct {
	Term         int
	LeaderID     string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Entry
	LeaderCommit int
}

// ReqVote represents the request the condidate sends to all the other servers when it doesn't
// receive the heartbeat Tick.
type ReqVote struct {
	Term         int    `json:"term"`
	CandidateID  string `json:"candidate_id"`
	LastLogIndex int    `json:"last_log_index"`
	LastLogTerm  int    `json:"last_log_term"`
}

// RequestVoteResponse represents the response to a requestVote RPC.
type RequestVoteResponse struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"vote_granted"`
	Reason      string
}

// ClientMessgaeResponse is for responding to an outside client
type ClientMessgaeResponse struct {
	Response string
}

// SafeDummyType is never meant to be used, and is a placeholder for args
type SafeDummyType struct {
	Data string
}
