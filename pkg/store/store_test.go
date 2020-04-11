package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/weavetest"

	"github.com/iov-one/block-metrics/pkg/models"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
	_ "github.com/lib/pq"
)

func TestLastBlock(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	if _, err := s.LatestBlock(ctx); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %q", err)
	}

	vID, err := s.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
	if err != nil {
		t.Fatalf("cannot create a validator: %s", err)
	}

	for i := 5; i < 100; i += 20 {
		block := models.Block{
			Height: int64(i),
			Hash:   hex.EncodeToString([]byte{0, 1, byte(i)}),
			// Postgres TIMESTAMPTZ precision is microseconds.
			Time:           time.Now().UTC().Round(time.Microsecond),
			ProposerID:     vID,
			ParticipantIDs: []int64{vID},
			Messages:       []string{"test/mymsg"},
		}
		if err := s.InsertBlock(ctx, block); err != nil {
			t.Fatalf("cannot insert block: %s", err)
		}

		got, err := s.LatestBlock(ctx)
		if err != nil {
			t.Fatalf("cannot get latest block: %s", err)
		}

		if !reflect.DeepEqual(got, &block) {
			t.Logf(" got %#v", got)
			t.Logf("want %#v", &block)
			t.Fatal("unexpected result")
		}
	}
}

func TestLoadBlockByHash(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	if _, err := s.LatestBlock(ctx); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %q", err)
	}

	vID, err := s.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
	if err != nil {
		t.Fatalf("cannot create a validator: %s", err)
	}

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

		got, err := s.LoadBlockByHash(ctx, hash)
		if err != nil {
			t.Fatalf("cannot get block by hash: %s", err)
		}

		if !reflect.DeepEqual(got, &block) {
			t.Logf(" got %#v", got)
			t.Logf("want %#v", &block)
			t.Fatal("unexpected result")
		}
	}
}

func TestLoadTx(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	if _, err := s.LatestBlock(ctx); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %q", err)
	}

	vID, err := s.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
	if err != nil {
		t.Fatalf("cannot create a validator: %s", err)
	}

	transactionsHashes := []string{
		"4311a5cb5e4f59de2ef3077363581dc4d53f86d0a69016a38e8e9a58b4cfb40f",
		"d926c9b781714898181a0b32c068ec24304dd9be045071f9a8655720efdfd282",
		"a2bc7dfba6813392c862c5bae1033f9fe8d36a7652efee03085e13a0908450b2",
		"2e49e9cabbd925937e15d00f5ba0c181146e2118e6f4dfb3d5e3d5e5e4958cfc",
		"01941c102d0d09796e5cc83c491c812f75e35798367c2c6e85f9360038441f54",
	}

	var txs []models.Transaction

	for i, txh := range transactionsHashes {
		tx := models.Transaction{
			Hash:    txh,
			BlockID: int64(i),
			Message: json.RawMessage(`{"path": "cash/send", "details": {"memo": "Lambo payment.", "amount": {"whole": 1, "ticker": "IOV"}, "source": "1cmvug0h4wsstwl42994z8c8uewcxk3f58syuj9", "destination": "1a9duw7yyxdfh8mrjxmuc0slu8a48muvx5mjvuc"}}`),
		}
		txs = append(txs, tx)
	}

	for i, tx := range txs {
		block := models.Block{
			Height: int64(i),
			Hash:   hex.EncodeToString([]byte{0, 1, byte(5 + i*20)}),
			// Postgres TIMESTAMPTZ precision is microseconds.
			Time:           time.Now().UTC().Round(time.Microsecond),
			ProposerID:     vID,
			ParticipantIDs: []int64{vID},
			Messages:       []string{"test/mymsg"},
			Transactions:   []models.Transaction{tx},
		}
		if err := s.InsertBlock(ctx, block); err != nil {
			t.Fatalf("cannot insert block: %s", err)
		}
	}

	for i, tx := range txs {
		got, err := s.LoadTx(ctx, transactionsHashes[i])
		if err != nil {
			t.Fatalf("cannot retrieve tx: %s", err)
		}
		if !reflect.DeepEqual(&tx, got) {
			t.Logf(" got %#v", got)
			t.Logf("want %#v", &tx)
			t.Fatal("unexpected result")
		}
	}
}

func TestLastNBlock(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	if _, err := s.LastNBlock(ctx, 1, 0); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %q", err)
	}

	vID, err := s.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
	if err != nil {
		t.Fatalf("cannot create a validator: %s", err)
	}

	// create and insert blocks
	var blocks []*models.Block
	for i := 5; i < 100; i += 20 {
		block := models.Block{
			Height: int64(i),
			Hash:   hex.EncodeToString([]byte{0, 1, byte(i)}),
			// Postgres TIMESTAMPTZ precision is microseconds.
			Time:           time.Now().UTC().Round(time.Microsecond),
			ProposerID:     vID,
			ParticipantIDs: []int64{vID},
			Messages:       []string{"test/mymsg"},
		}
		if err := s.InsertBlock(ctx, block); err != nil {
			t.Fatalf("cannot insert block: %s", err)
		}

		blocks = append(blocks, &block)
	}

	got, err := s.LastNBlock(ctx, 5, 0)
	assert.Nil(t, err)

	for i, g := range got {
		expected := blocks[len(blocks)-i-1]
		if !reflect.DeepEqual(g, expected) {
			t.Logf(" got %#v", g)
			t.Logf("want %#v", expected)
			t.Fatal("unexpected result")
		}
	}

	_, err = s.LastNBlock(ctx, 101, 0)
	if !ErrLimit.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestStoreInsertValidator(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	pubkeyA := []byte{0x01, 0, 0xbe, 'a'}
	addrA := []byte{0x02, 'a'}
	if _, err := s.InsertValidator(ctx, pubkeyA, addrA); err != nil {
		t.Fatalf("cannot create 'a' validator: %s", err)
	}

	pubkeyB := []byte{0x01, 0, 0xbe, 'b'}
	addrB := []byte{0x02, 'b'}
	if _, err := s.InsertValidator(ctx, pubkeyB, addrB); err != nil {
		t.Fatalf("cannot create 'b' validator: %s", err)
	}

	if _, err := s.InsertValidator(ctx, pubkeyA, []byte{0x99}); !ErrConflict.Is(err) {
		t.Fatalf("was able to create a validator with an existing public key: %q", err)
	}
	if _, err := s.InsertValidator(ctx, []byte{0x99}, addrA); !ErrConflict.Is(err) {
		t.Fatalf("was able to create a validator with an existing address: %q", err)
	}
}

func TestStoreInsertBlock(t *testing.T) {
	type validator struct {
		address []byte
		pubkey  []byte
	}
	cases := map[string]struct {
		validators []validator
		block      models.Block
		wantErr    *errors.Error
	}{
		"success": {
			validators: []validator{
				{address: []byte{0x01}, pubkey: []byte{0x01, 0, 0x01}},
				{address: []byte{0x02}, pubkey: []byte{0x02, 0, 0x02}},
				{address: []byte{0x03}, pubkey: []byte{0x03, 0, 0x03}},
			},
			block: models.Block{
				Height:         1,
				Hash:           hex.EncodeToString([]byte{0, 1, 2, 3}),
				Time:           time.Now().UTC().Round(time.Millisecond),
				ProposerID:     2,
				ParticipantIDs: []int64{2, 3},
				Messages:       []string{"test/one"},
			},
		},
		"success with one missing": {
			validators: []validator{
				{address: []byte{0x01}, pubkey: []byte{0x01, 0, 0x01}},
				{address: []byte{0x02}, pubkey: []byte{0x02, 0, 0x02}},
				{address: []byte{0x03}, pubkey: []byte{0x03, 0, 0x03}},
			},
			block: models.Block{
				Height:         2,
				Hash:           hex.EncodeToString([]byte{0, 1, 2, 3}),
				Time:           time.Now().UTC().Round(time.Millisecond),
				ProposerID:     3,
				ParticipantIDs: []int64{2, 3},
				MissingIDs:     []int64{1},
				Messages:       []string{"test/one", "test/two"},
			},
		},
		"missing participant ids": {
			validators: []validator{
				{address: []byte{0x01}, pubkey: []byte{0x01, 0, 0x01}},
			},
			block: models.Block{
				Height:         1,
				Hash:           hex.EncodeToString([]byte{0, 1, 2, 3}),
				Time:           time.Now().UTC().Round(time.Millisecond),
				ProposerID:     1,
				ParticipantIDs: nil,
				Messages:       []string{},
			},
			wantErr: ErrConflict,
		},
		"invalid proposer ID": {
			validators: []validator{
				{address: []byte{0x01}, pubkey: []byte{0x01, 0, 0x01}},
				{address: []byte{0x02}, pubkey: []byte{0x02, 0, 0x02}},
				{address: []byte{0x03}, pubkey: []byte{0x03, 0, 0x03}},
			},
			block: models.Block{
				Height:         1,
				Hash:           hex.EncodeToString([]byte{0, 1, 2, 3}),
				Time:           time.Now().UTC().Round(time.Millisecond),
				ProposerID:     4,
				ParticipantIDs: []int64{2, 3},
				Messages:       []string{},
			},
			wantErr: ErrConflict,
		},
		// This is not implemented.
		//
		// "invalid participant ids ID": {
		// 	validators: []validator{
		// 		{address: []byte{0x01}, pubkey: []byte{0x01, 0, 0x01}},
		// 		{address: []byte{0x02}, pubkey: []byte{0x02, 0, 0x02}},
		// 		{address: []byte{0x03}, pubkey: []byte{0x03, 0, 0x03}},
		// 	},
		// 	block: Block{
		// 		Height:         1,
		// 		Hash:           []byte{0, 1, 2, 3},
		// 		Time:           time.Now().UTC().Round(time.Millisecond),
		// 		ProposerID:     2,
		// 		ParticipantIDs: []int64{666, 999},
		// 	},
		// 	wantErr: ErrConflict,
		// },
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db, cleanup := EnsureDB(t)
			defer cleanup()

			ctx := context.Background()
			s := NewStore(db)

			for _, v := range tc.validators {
				if _, err := s.InsertValidator(ctx, v.pubkey, v.address); err != nil {
					t.Fatalf("cannot ensure validator: %s", err)
				}
			}

			if err := s.InsertBlock(ctx, tc.block); !tc.wantErr.Is(err) {
				t.Fatalf("want %q error, got %q", tc.wantErr, err)
			}

			if tc.wantErr == nil {
				// ensure we can load it back the same
				loaded, err := s.LoadBlock(ctx, tc.block.Height)
				if err != nil {
					t.Fatalf("cannot re-load block %v", err)
				}
				if !reflect.DeepEqual(loaded, &tc.block) {
					t.Logf(" got %#v", loaded)
					t.Logf("want %#v", &tc.block)
					t.Fatal("unexpected result")
				}
			}
		})
	}
}

func TestStoreAccount(t *testing.T) {
	db, cleanup := EnsureDB(t)
	defer cleanup()

	ctx := context.Background()

	s := NewStore(db)

	targets := []account.BlockchainAddress{
		{
			BlockchainID: "cosmos",
			Address:      "test",
		},
		{
			BlockchainID: "cosmos1",
			Address:      "test1",
		},
	}
	msg := account.RegisterAccountMsg{
		Domain:  "domain",
		Name:    "name",
		Owner:   weavetest.NewCondition().Address(),
		Targets: targets,
	}

	if err := s.InsertAccount(ctx, &msg); err != nil {
		t.Fatalf("cannot insert account: %s", err)
	}

	acc, err := s.LoadAccount(ctx, "name", "domain")
	if err != nil {
		t.Fatalf("cannot load account: %s", err)
	}

	if acc.Domain != msg.Domain || acc.Name != msg.Name || acc.Owner != msg.Owner.String() {
		t.Logf("expected: %+v", msg)
		t.Fatalf("got: %+v", acc)
	}

	accTargets, err := s.LoadAccountTargets(ctx, "name", "domain")
	if err != nil {
		t.Fatalf("cannot load account: %s", err)
	}

	t.Logf("sent account targets: %+v", targets)
	t.Logf("got account targets: %+v", accTargets)

	newTargets := []account.BlockchainAddress{
		{
			BlockchainID: "new",
			Address:      "new",
		},
		{
			BlockchainID: "new1",
			Address:      "new1",
		},
	}
	replaceMsg := account.ReplaceAccountTargetsMsg{
		Domain:     "domain",
		Name:       "name",
		NewTargets: newTargets,
	}
	if err := s.ReplaceAccountTargets(ctx, &replaceMsg); err != nil {
		t.Fatalf("cannot replace account: %s", err)
	}

	accTargets, err = s.LoadAccountTargets(ctx, "name", "domain")
	if err != nil {
		t.Fatalf("cannot load account: %s", err)
	}

	t.Logf("sent account targets: %+v", targets)
	t.Logf("got account targets: %+v", accTargets)

}
