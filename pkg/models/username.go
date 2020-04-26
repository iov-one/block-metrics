package models

type UsernameTarget struct {
	ID           int64  `json:"-"`
	Username     int64  `json:"name"`
	BlockchainID string `json:"blockchain_id"`
	Address      string `json:"address"`
}
