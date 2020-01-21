package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/iov-one/block-metrics/pkg/config"
	"github.com/iov-one/block-metrics/pkg/metrics"
	"github.com/iov-one/block-metrics/pkg/store"

	"github.com/iov-one/weave/errors"
)

func main() {
	conf := config.Configuration{
		DBHost:          os.Getenv("POSTGRES_HOST"),
		DBName:          os.Getenv("POSTGRES_DB_NAME"),
		DBUser:          os.Getenv("POSTGRES_USER"),
		DBPass:          os.Getenv("POSTGRES_PASSWORD"),
		DBSSL:           os.Getenv("POSTGRES_SSL_ENABLE"),
		TendermintWsURI: os.Getenv("TENDERMINT_WS_URI"),
		Hrp:             os.Getenv("HRP"),
	}

	if err := run(conf); err != nil {
		log.Fatal(err)
	}
}

func run(conf config.Configuration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbUri := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", conf.DBUser, conf.DBPass,
		conf.DBHost, conf.DBName, conf.DBSSL)
	db, err := sql.Open("postgres", dbUri)
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
