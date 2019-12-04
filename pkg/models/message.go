package models

import (
	"encoding/json"
)

type Message struct {
	Path      string          `json:"path"`
	Details   json.RawMessage `json:"details"`
	Multisigs []string        `json:"multisig_contract_ids,omitempty"`
}
