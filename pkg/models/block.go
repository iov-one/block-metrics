package models

import (
	"time"
)

type Block struct {
	Height         int64         `json:"height"`
	Hash           string        `json:"hash"`
	Time           time.Time     `json:"time"`
	ProposerID     int64         `json:"-"`
	ProposerName   string        `json:"proposer_name"`
	ParticipantIDs []int64       `json:"-"`
	MissingIDs     []int64       `json:"-"`
	Messages       []string      `json:"messages,omitempty"`
	FeeFrac        uint64        `json:"fee_frac"`
	Transactions   []Transaction `json:"transactions"`
}
