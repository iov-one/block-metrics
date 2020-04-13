package app

import (
	"context"
	"net/http"
	"time"

	"github.com/iov-one/block-metrics/cmd/api/app/handlers"
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

	blocksHandler := handlers.BlocksHandler{Store: store}
	blockApi := g.Group("/blocks")
	blockApi.GET("", blocksHandler.GetBlocks)
	blockApi.GET("/latest", blocksHandler.GetLatestBlock)
	blockApi.GET("/last/:key", blocksHandler.GetLatestNBlocks)
	blockApi.GET("/hash/:hash", blocksHandler.GetBlockByHash)
	blockApi.GET("/height/:height", blocksHandler.GetBlockByHeight)

	txsHandler := handlers.TxsHandler{Store: store}
	txApi := g.Group("/txs")
	txApi.GET("/latest", txsHandler.GetLatestTX)
	txApi.GET("/last/:number", txsHandler.GetLastNTx)
	txApi.GET("/hash/:hash", txsHandler.GetTx)
	txApi.POST("/query", txsHandler.QueryTxsByParams)

	accountsHandler := handlers.AccountsHandler{Store: store}
	accountApi := g.Group("/accounts")
	accountApi.GET("", accountsHandler.GetAccount)

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
