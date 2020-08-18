package handlers

import (
	"net/http"
	"strconv"

	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/labstack/echo/v4"
)

type BlocksHandler struct {
	Store *store.Store
}

// e.GET("/blocks/latest", a.getLatestBlock)
func (h *BlocksHandler) GetLatestBlock(ctx echo.Context) error {
	block, err := h.Store.LatestBlock(ctx.Request().Context())
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, block)
}

// e.GET("/blocks/latest/:key", a.getLatestNBlocks)
func (h *BlocksHandler) GetLatestNBlocks(c echo.Context) error {
	key := c.Param("key")
	n, err := strconv.Atoi(key)
	if err != nil {
		return err
	}
	// get latest n blocks from db
	block, err := h.Store.LastNBlock(c.Request().Context(), n, 0)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, block)
}

type blockQuery struct {
	Limit int `txQuery:"limit"`
	// After is block height
	After int `txQuery:"after"`
}

// e.GET("/blocks?Limit=:Limit&After=:After", a.getLatestNBlocks)
func (h *BlocksHandler) GetBlocks(c echo.Context) error {
	q := new(blockQuery)
	q.Limit = 10
	q.After = 0
	if err := c.Bind(q); err != nil {
		return err
	}
	// get latest n blocks from db
	block, err := h.Store.LastNBlock(c.Request().Context(), q.Limit, q.After)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, block)
}

// e.GET("/blocks/hash/:hash", a.getBlockByHash)
func (h *BlocksHandler) GetBlockByHash(c echo.Context) error {
	hash := c.Param("hash")

	// get latest block from db
	block, err := h.Store.LoadBlockByHash(c.Request().Context(), hash)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, block)
}

// e.GET("/blocks/height/:height", a.getBlockByHeight)
func (h *BlocksHandler) GetBlockByHeight(c echo.Context) error {
	height := c.Param("height")

	// get block from db by block height
	block, err := h.Store.LoadBlockByHeight(c.Request().Context(), height)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, block)
}
