package models

import (
	"encoding/json"
)

type Transaction struct {
	Hash    string          `json:"hash"`
	BlockID int64           `json:"block_height"`
	Message json.RawMessage `json:"message,omitempty"`
}
