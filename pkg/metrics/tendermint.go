package metrics

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iov-one/block-metrics/pkg/errors"
	"github.com/gorilla/websocket"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
)

type TendermintClient struct {
	idCnt uint64

	conn *websocket.Conn

	stop chan struct{}

	mu   sync.Mutex
	resp map[string]chan<- *jsonrpcResponse
}

// DialTendermint returns a client that is maintains a websocket connection to
// tendermint API. The websocket is used instead of standard HTTP connection to
// lower the latency, bypass throttling and to allow subscription requests.
func DialTendermint(websocketURL string) (*TendermintClient, error) {
	c, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}
	cli := &TendermintClient{
		conn: c,
		stop: make(chan struct{}),
		resp: make(map[string]chan<- *jsonrpcResponse),
	}
	go cli.readLoop()
	return cli, nil
}

func (c *TendermintClient) Close() error {
	close(c.stop)
	return c.conn.Close()
}

func (c *TendermintClient) readLoop() {
	for {
		select {
		case <-c.stop:
			return
		default:
		}

		var resp jsonrpcResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			log.Printf("cannot unmarshal JSONRPC message: %s", err)
			continue
		}

		c.mu.Lock()
		respc, ok := c.resp[resp.CorrelationID]
		delete(c.resp, resp.CorrelationID)
		c.mu.Unlock()

		if ok {
			// repc is expected to be a buffered channel so this
			// operation must never block.
			respc <- &resp
		}
	}
}

// Do makes a jsonrpc call. This method is safe for concurrent calls.
//
// Use API as described in https://tendermint.com/rpc/
func (c *TendermintClient) Do(method string, dest interface{}, args ...interface{}) error {
	params := make([]string, len(args))
	for i, v := range args {
		params[i] = fmt.Sprint(v)
	}
	req := jsonrpcRequest{
		ProtocolVersion: "2.0",
		CorrelationID:   fmt.Sprint(atomic.AddUint64(&c.idCnt, 1)),
		Method:          method,
		Params:          params,
	}

	respc := make(chan *jsonrpcResponse, 1)
	c.mu.Lock()
	c.resp[req.CorrelationID] = respc
	c.mu.Unlock()

	if err := c.conn.WriteJSON(req); err != nil {
		return errors.Wrap(err, "write JSON")
	}

	resp := <-respc

	if resp.Error != nil {
		return errors.Wrapf(ErrFailedResponse,
			"%d: %s",
			resp.Error.Code, resp.Error.Message)
	}
	if err := json.Unmarshal(resp.Result, dest); err != nil {
		return errors.Wrap(err, "cannot unmarshal result")
	}
	return nil
}

type jsonrpcRequest struct {
	ProtocolVersion string   `json:"jsonrpc"`
	CorrelationID   string   `json:"id"`
	Method          string   `json:"method"`
	Params          []string `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	ProtocolVersion string `json:"jsonrpc"`
	CorrelationID   string `json:"id"`
	Result          json.RawMessage
	Error           *struct {
		Code    int64
		Message string
	}
}

var (
	ErrFailedResponse = errors.New("failed response")
)

// AbciInfo returns abci_info.
func AbciInfo(c *TendermintClient) (*ABCIInfo, error) {
	var payload struct {
		Response struct {
			LastBlockHeight sint64 `json:"last_block_height"`
		} `json:"response"`
	}

	if err := c.Do("abci_info", &payload); err != nil {
		return nil, errors.Wrap(err, "query tendermint")
	}

	return &ABCIInfo{LastBlockHeight: int64(payload.Response.LastBlockHeight)}, nil
}

type ABCIInfo struct {
	LastBlockHeight int64 `json:"last_block_height"`
}

// Validators return all validators as represented on the block at given
// height.
func Validators(ctx context.Context, c *TendermintClient, blockHeight int64) ([]*TendermintValidator, error) {
	var payload struct {
		Validators []struct {
			Address hexstring
			PubKey  struct {
				Value []byte
			} `json:"pub_key"`
		}
	}
	if err := c.Do("validators", &payload, blockHeight); err != nil {
		return nil, errors.Wrap(err, "query tendermint")
	}
	var validators []*TendermintValidator
	for _, v := range payload.Validators {
		validators = append(validators, &TendermintValidator{
			Address: v.Address,
			PubKey:  v.PubKey.Value,
		})
	}
	return validators, nil
}

type TendermintValidator struct {
	Address []byte
	PubKey  []byte
}

// ValidatorAddresses extracts just the addresses of out a signing set
func ValidatorAddresses(validators []*TendermintValidator) [][]byte {
	res := make([][]byte, len(validators))
	for i, v := range validators {
		res[i] = v.Address
	}
	return res
}

// SubtractSets returns all elements in a that are not in b
func SubtractSets(a [][]byte, b [][]byte) [][]byte {
	var res [][]byte
	// splice out all those who we find
	for _, check := range a {
		if !contains(b, check) {
			res = append(res, check)
		}
	}
	return res
}

func contains(haystack [][]byte, needle []byte) bool {
	for _, hay := range haystack {
		if bytes.Equal(hay, needle) {
			return true
		}
	}
	return false
}

func Commit(ctx context.Context, c *TendermintClient, height int64) (*TendermintCommit, error) {
	var payload struct {
		SignedHeader struct {
			Header struct {
				Height          sint64    `json:"height"`
				Time            time.Time `json:"time"`
				ProposerAddress hexstring `json:"proposer_address"`
				ValidatorsHash  hexstring `json:"validators_hash"`
			} `json:"header"`
			Commit struct {
				BlockID struct {
					Hash hexstring `json:"hash"`
				} `json:"block_id"`
				Precommits []*struct {
					ValidatorAddress hexstring `json:"validator_address"`
				} `json:"precommits"`
			} `json:"commit"`
		} `json:"signed_header"`
	}

	if err := c.Do("commit", &payload, height); err != nil {
		return nil, errors.Wrap(err, "query tendermint")
	}

	commit := TendermintCommit{
		Height:          payload.SignedHeader.Header.Height.Int64(),
		Hash:            payload.SignedHeader.Commit.BlockID.Hash,
		Time:            payload.SignedHeader.Header.Time.UTC(),
		ProposerAddress: payload.SignedHeader.Header.ProposerAddress,
		ValidatorsHash:  payload.SignedHeader.Header.ValidatorsHash,
	}

	for _, pc := range payload.SignedHeader.Commit.Precommits {
		if pc == nil {
			continue
		}
		commit.ParticipantAddresses = append(commit.ParticipantAddresses, pc.ValidatorAddress)
	}

	return &commit, nil
}

type TendermintCommit struct {
	Height               int64
	Hash                 []byte
	Time                 time.Time
	ProposerAddress      []byte
	ValidatorsHash       []byte
	ParticipantAddresses [][]byte
}

func FetchBlock(ctx context.Context, c *TendermintClient, height int64) (*TendermintBlock, error) {
	var payload struct {
		Block struct {
			Header struct {
				Height sint64    `json:"height"`
				Time   time.Time `json:"time"`
			} `json:"header"`
			Data struct {
				Txs [][]byte `json:"txs"`
			} `json:"data"`
		} `json:"block"`
	}

	if err := c.Do("block", &payload, height); err != nil {
		return nil, errors.Wrap(err, "query tendermint")
	}

	block := TendermintBlock{
		Height: payload.Block.Header.Height.Int64(),
		Time:   payload.Block.Header.Time.UTC(),
	}

	for _, rawTx := range payload.Block.Data.Txs {
		var tx bnsd.Tx
		if err := tx.Unmarshal(rawTx); err != nil {
			return nil, errors.Wrap(err, "cannot unmarshal transaction")
		}
		block.Transactions = append(block.Transactions, &tx)
		block.TransactionHashes = append(block.TransactionHashes, sha256.Sum256(rawTx))
	}

	return &block, nil
}

type TendermintBlock struct {
	Height            int64
	Time              time.Time
	Transactions      []*bnsd.Tx
	TransactionHashes [][32]byte
}
