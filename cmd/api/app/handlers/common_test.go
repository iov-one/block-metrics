package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/iov-one/block-metrics/pkg/models"
	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/weavetest"
)

func prepareStore(t *testing.T, ctx context.Context) (str *store.Store, cleanup func()) {
	t.Helper()
	testdb, cleanup := store.EnsureDB(t)
	s := store.NewStore(testdb)

	vID, err := s.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
	if err != nil {
		t.Fatalf("cannot create a validator: %s", err)
	}

	// create and insert blocks
	for i := 5; i < 100; i += 20 {
		hash := hex.EncodeToString([]byte{0, 1, byte(i)})
		txs := []models.Transaction{
			models.Transaction{
				Hash:    hex.EncodeToString([]byte{0, 1, 0, byte(i)}),
				BlockID: int64(i),
				Message: json.RawMessage(`{"path": "test/mymsg", "details": {"memo": "Hello world! Congratulations to the team in Osaka. Hurray", "amount": {"whole": 1, "ticker": "IOV"}, "source": "iov1ua6tdcyw8jddn5660qcx2ndhjp4skqk4dkurrl", "destination": "iov1c9eprq0gxdmwl9u25j568zj7ylqgc7aj2fw2xj"}}`),
			},
			models.Transaction{
				Hash:    hex.EncodeToString([]byte{0, 1, 0, byte(i + 400)}),
				BlockID: int64(i),
				Message: json.RawMessage(`{"path": "test/mymsg", "details": {"memo": "Hello world! Congratulations to the team in Osaka. Hurray", "amount": {"whole": 1, "ticker": "IOV"}, "source": "iov1ua6tdcyw8jddn5660qcx2ndhjp4skqk4dkurrl", "destination": "iov1c9eprq0gxdmwl9u25j568zj7ylqgc7aj2fw2xj"}}`),
			},
		}
		block := models.Block{
			Height: int64(i),
			Hash:   hash,
			// Postgres TIMESTAMPTZ precision is microseconds.
			Time:           time.Now().UTC().Round(time.Microsecond),
			ProposerID:     vID,
			ParticipantIDs: []int64{vID},
			Messages:       []string{"test/mymsg"},
			Transactions:   txs,
		}
		if err := s.InsertBlock(ctx, block); err != nil {
			t.Fatalf("cannot insert block: %s", err)
		}
	}

	// insert account
	targets := []account.BlockchainAddress{
		{BlockchainID: "cosmos", Address: weavetest.NewCondition().Address().String()},
		{BlockchainID: "lisk", Address: weavetest.NewCondition().Address().String()},
	}
	msg := account.RegisterAccountMsg{
		Domain:  "dtest",
		Name:    "ntest",
		Owner:   weavetest.NewCondition().Address(),
		Targets: targets,
	}
	if err := s.InsertAccount(ctx, &msg); err != nil {
		t.Fatalf("cannot insert account: %s", err)
	}

	targets = []account.BlockchainAddress{
		{BlockchainID: "btc", Address: weavetest.NewCondition().Address().String()},
		{BlockchainID: "ava", Address: weavetest.NewCondition().Address().String()},
	}
	msg = account.RegisterAccountMsg{
		Domain:  "orkun",
		Name:    "deli",
		Owner:   weavetest.NewCondition().Address(),
		Targets: targets,
	}
	if err := s.InsertAccount(ctx, &msg); err != nil {
		t.Fatalf("cannot insert account: %s", err)
	}

	return s, cleanup
}
