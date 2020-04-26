package models

type Username struct {
	ID   int64  `json:"-"`
	Name string `json:"name"`
}

type UsernameTarget struct {
	ID           int64  `json:"-"`
	UsernameID   int64  `json:"-"`
	BlockchainID string `json:"blockchain_id"`
	Address      string `json:"address"`
}
