package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"net/http/httptest"
	"testing"

	"github.com/iov-one/block-metrics/pkg/models"
	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestGetLatestBlock(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/blocks/latest", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app := App{}
	ctx := context.Background()
	ectx := e.NewContext(req, rec)

	store, cleanup := prepareStore(t, ctx)

	app.Initialize(ctx, store)

	defer cleanup()

	expected := `{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}`

	if assert.NoError(t, app.getLatestBlock(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}

func TestGetLatestNBlock(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app := App{}
	ctx := context.Background()
	ectx := e.NewContext(req, rec)
	ectx.SetPath("/blocks/latest/:key")
	ectx.SetParamNames("key")
	ectx.SetParamValues("2")

	store, cleanup := prepareStore(t, ctx)

	app.Initialize(ctx, store)

	defer cleanup()

	expected := `[{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null},{"Height":65,"Hash":"000141","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}]`

	if assert.NoError(t, app.getLatestNBlocks(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}

func TestGetBlockByHash(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app := App{}
	ctx := context.Background()
	ectx := e.NewContext(req, rec)
	ectx.SetPath("/blocks/hash/:hash")
	ectx.SetParamNames("hash")
	ectx.SetParamValues("1010101010101010")

	store, cleanup := prepareStore(t, ctx)

	app.Initialize(ctx, store)

	defer cleanup()

	expected := `{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}`

	if assert.NoError(t, app.getBlockByHash(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}

func prepareStore(t *testing.T, ctx context.Context) (s *store.Store, cleanup func()) {
	testdb, cleanup := store.EnsureDB(t)
	str := store.NewStore(testdb)

	vID, err := str.InsertValidator(ctx, []byte{0x01, 0, 0xbe, 'a'}, []byte{0x02})
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
		if err := str.InsertBlock(ctx, block); err != nil {
			t.Fatalf("cannot insert block: %s", err)
		}
	}

	return str, cleanup
}
