package main

type ConsensusState struct {
	currentBlockID         int
	receivedMinLeaderValue int
	ownLeaderValue         int
}
