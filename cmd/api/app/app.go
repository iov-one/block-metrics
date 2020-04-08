package app

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

type App struct {
	Server *echo.Echo
	Store  *store.Store
	ctx    context.Context
}

func (a *App) Initialize(ctx context.Context, store *store.Store) {
	a.ctx = ctx
	a.Store = store

	e := echo.New()
	g := e.Group("/api")

	blockApi := g.Group("/blocks")
	blockApi.GET("", a.getBlocks)
	blockApi.GET("/latest", a.getLatestBlock)
	blockApi.GET("/last/:key", a.getLatestNBlocks)
	blockApi.GET("/hash/:hash", a.getBlockByHash)
	blockApi.GET("/height/:height", a.getBlockByHeight)

	txApi := g.Group("/txs")
	txApi.GET("/latest", a.getLatestTX)
	txApi.GET("/last/:number", a.getLastNTx)
	txApi.GET("/hash/:hash", a.getTx)
	txApi.POST("/query", a.queryTxsByParams)

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.RequestID())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		StackSize:       1 << 10, // 1 KB
	}))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		httpCode := http.StatusInternalServerError
		err = &echo.HTTPError{
			Code:     httpCode,
			Message:  err.Error(),
			Internal: err,
		}
		e.DefaultHTTPErrorHandler(err, c)
	}
	a.Server = e
}

func (a *App) Run(ctx context.Context, port string) {
	go func() {
		if err := a.Server.Start(":" + port); err != nil {
			a.Server.Logger.Info("Shutting down the server")
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.Server.Shutdown(ctx); err != nil {
		a.Server.Logger.Fatal(err)
	}
}

// e.GET("/blocks/latest", a.getLatestBlock)
func (a *App) getLatestBlock(ctx echo.Context) error {
	block, err := a.Store.LatestBlock(a.ctx)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, block)
}

// e.GET("/blocks/latest/:key", a.getLatestNBlocks)
func (a *App) getLatestNBlocks(c echo.Context) error {
	key := c.Param("key")
	n, err := strconv.Atoi(key)
	if err != nil {
		return err
	}
	// get latest n blocks from db
	block, err := a.Store.LastNBlock(a.ctx, n, 0)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, block)
}

type blockQuery struct {
	Limit int `query:"limit"`
	// After is block height
	After int `query:"after"`
}

// e.GET("/blocks?Limit=:Limit&After=:After", a.getLatestNBlocks)
func (a *App) getBlocks(c echo.Context) error {
	q := new(blockQuery)
	q.Limit = 10
	q.After = 0
	if err := c.Bind(q); err != nil {
		return err
	}
	// get latest n blocks from db
	block, err := a.Store.LastNBlock(a.ctx, q.Limit, q.After)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, block)
}

// e.GET("/blocks/hash/:hash", a.getBlockByHash)
func (a *App) getBlockByHash(c echo.Context) error {
	hash := c.Param("hash")

	// get latest block from db
	block, err := a.Store.LoadBlockByHash(a.ctx, hash)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, block)
}

// e.GET("/blocks/height/:height", a.getBlockByHeight)
func (a *App) getBlockByHeight(c echo.Context) error {
	height := c.Param("height")

	// get block from db by block height
	block, err := a.Store.LoadBlockByHeight(a.ctx, height)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, block)
}

// e.GET("/txs/latest", a.getLatestTX)
func (a *App) getLatestTX(ctx echo.Context) error {
	tx, err := a.Store.LoadLatestNTx(a.ctx, 1)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, tx)
}

// e.GET("/txs/last/:number", a.getLastNTx)
func (a *App) getLastNTx(ctx echo.Context) error {
	number := ctx.Param("number")
	n, err := strconv.Atoi(number)
	if err != nil {
		return err
	}

	tx, err := a.Store.LoadLatestNTx(a.ctx, n)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, tx)
}

// e.GET("/txs/{hash:[0-9a-fA-F]+}", a.getTx)
func (a *App) getTx(c echo.Context) error {
	hash := c.Param("hash")

	// get latest block from db
	tx, err := a.Store.LoadTx(a.ctx, hash)
	if err != nil {
		return err
	}
	// convert to json
	return c.JSON(http.StatusOK, tx)
}

type query struct {
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	Memo        string `json:"memo,omitempty"`
}

func (a *App) queryTxsByParams(c echo.Context) error {
	q := new(query)
	if err := c.Bind(q); err != nil {
		return err
	}

	txs, err := a.Store.LoadTxsByParams(a.ctx, q.Source, q.Destination, q.Memo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, txs)
}
