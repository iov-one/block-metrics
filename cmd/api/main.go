package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"

	"github.com/iov-one/block-metrics/cmd/api/app"
	"github.com/iov-one/block-metrics/pkg/store"
)

func main() {
	conf := Configuration{
		DBHost:         os.Getenv("POSTGRES_HOST"),
		DBName:         os.Getenv("POSTGRES_DB_NAME"),
		DBUser:         os.Getenv("POSTGRES_USER"),
		DBPass:         os.Getenv("POSTGRES_PASSWORD"),
		DBSSL:          os.Getenv("POSTGRES_SSL_ENABLE"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		Port:           os.Getenv("PORT"),
	}

	ctx, cancel := context.WithCancel(context.Background())

	dbUri := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", conf.DBUser, conf.DBPass,
		conf.DBHost, conf.DBName, conf.DBSSL)

	db, err := sql.Open("postgres", dbUri)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer db.Close()

	if err := store.EnsureSchema(db); err != nil {
		panic(err)
	}

	a := app.App{}
	st := store.NewStore(db)
	a.Initialize(ctx, st)

	go func() {
		defer cancel()
		quit := make(chan os.Signal)
		signal.Notify(quit, os.Interrupt)
		<-quit
	}()

	a.Run(ctx, conf.Port)
}
