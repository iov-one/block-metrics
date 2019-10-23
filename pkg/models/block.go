package models

import (
	"time"
)

type Block struct {
	Height         int64
	Hash           []byte
	Time           time.Time
	ProposerID     int64
	ParticipantIDs []int64
	MissingIDs     []int64
	Messages       []string
	FeeFrac        uint64
	Transactions   []Transaction
}
