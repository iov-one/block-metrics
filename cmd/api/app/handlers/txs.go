package handlers

import (
	"net/http"
	"strconv"

	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/labstack/echo/v4"
)

type TxsHandler struct {
	Store *store.Store
}

// e.GET("/txs/latest", a.getLatestTX)
func (h *TxsHandler) GetLatestTX(c echo.Context) error {
	tx, err := h.Store.LoadLatestNTx(c.Request().Context(), 1)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tx)
}

// e.GET("/txs/last/:number", a.getLastNTx)
func (h *TxsHandler) GetLastNTx(c echo.Context) error {
	number := c.Param("number")
	n, err := strconv.Atoi(number)
	if err != nil {
		return err
	}

	tx, err := h.Store.LoadLatestNTx(c.Request().Context(), n)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tx)
}

// e.GET("/txs/{hash:[0-9a-fA-F]+}", a.getTx)
func (h *TxsHandler) GetTx(c echo.Context) error {
	hash := c.Param("hash")

	// get latest block from db
	tx, err := h.Store.LoadTx(c.Request().Context(), hash)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, tx)
}

type txQuery struct {
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	Memo        string `json:"memo,omitempty"`
}

func (h *TxsHandler) QueryTxsByParams(c echo.Context) error {
	q := new(txQuery)
	if err := c.Bind(q); err != nil {
		return err
	}

	txs, err := h.Store.LoadTxsByParams(c.Request().Context(), q.Source, q.Destination, q.Memo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, txs)
}
