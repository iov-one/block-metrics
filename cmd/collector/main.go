package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/iov-one/block-metrics/pkg/config"
	"github.com/iov-one/block-metrics/pkg/metrics"
	"github.com/iov-one/block-metrics/pkg/store"

	"github.com/iov-one/weave/errors"
)

func main() {
	conf := config.Configuration{
		PostgresURI:     os.Getenv("DATABASE_URL"),
		TendermintWsURI: os.Getenv("TENDERMINT_WS_URI"),
		Hrp:             os.Getenv("HRP"),
	}

	if err := run(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(conf config.Configuration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sql.Open("postgres", conf.PostgresURI)
	if err != nil {
		return fmt.Errorf("cannot connect to postgres: %s", err)
	}
	defer db.Close()

	if err := store.EnsureSchema(db); err != nil {
		return fmt.Errorf("ensure schema: %s", err)
	}

	st := store.NewStore(db)

	tmc, err := metrics.DialTendermint(conf.TendermintWsURI)
	if err != nil {
		return errors.Wrap(err, "dial tendermint")
	}
	defer tmc.Close()

	inserted, err := metrics.Sync(ctx, tmc, st, conf.Hrp)
	if err != nil {
		return errors.Wrap(err, "sync")
	}

	fmt.Println("inserted:", inserted)

	return nil
}
