package models

type Account struct {
	ID     int64  `json:"-"`
	Domain string `json:"domain"`
	Name   string `json:"name"`
	Owner  string `json:"owner"`
	Broker string `json:"broker"`
}

type AccountTarget struct {
	ID           int64  `json:"-"`
	AccountID    int64  `json:"-"`
	BlockchainID string `json:"blockchain_id"`
	Address      string `json:"address"`
}
