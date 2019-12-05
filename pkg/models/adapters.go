package models

import (
	coin "github.com/iov-one/weave/coin"
)

type CashSendMsgAdapter struct {
	Source      string     `json:"source"`
	Destination string     `json:"destination"`
	Amount      *coin.Coin `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`
	// max length 128 character
	Memo string `json:"memo"`
}
