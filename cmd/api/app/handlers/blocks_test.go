package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetLatestBlock(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/blocks/latest", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	ctx := context.Background()
	ectx := e.NewContext(req, rec)

	store, cleanup := prepareStore(t, ctx)
	h := BlocksHandler{Store: store}

	defer cleanup()

	expected := `{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}`

	if assert.NoError(t, h.GetLatestBlock(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}

func TestGetLatestNBlock(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	ctx := context.Background()
	ectx := e.NewContext(req, rec)
	ectx.SetPath("/blocks/latest/:key")
	ectx.SetParamNames("key")
	ectx.SetParamValues("2")

	store, cleanup := prepareStore(t, ctx)
	h := BlocksHandler{Store: store}

	defer cleanup()

	expected := `[{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null},{"Height":65,"Hash":"000141","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}]`

	if assert.NoError(t, h.GetLatestNBlocks(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}

func TestGetBlockByHash(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	ctx := context.Background()
	ectx := e.NewContext(req, rec)
	ectx.SetPath("/blocks/hash/:hash")
	ectx.SetParamNames("hash")
	ectx.SetParamValues("1010101010101010")

	store, cleanup := prepareStore(t, ctx)
	h := BlocksHandler{Store: store}

	defer cleanup()

	expected := `{"Height":85,"Hash":"000155","Time":"2009-11-17T20:34:58.651387Z","ProposerID":1,"ParticipantIDs":[1],"MissingIDs":null,"Messages":["test/mymsg"],"FeeFrac":0,"Transactions":null}`

	if assert.NoError(t, h.GetBlockByHash(ectx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expected+"\n", rec.Body.String())
	}
}
