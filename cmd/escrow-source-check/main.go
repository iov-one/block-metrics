package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/iov-one/block-metrics/pkg/config"
	"github.com/iov-one/block-metrics/pkg/store"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.Configuration{
		PostgresURI: os.Getenv("DATABASE_URL"),
		Hrp:         os.Getenv("HRP"),
	}

	db, err := sql.Open("postgres", conf.PostgresURI)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer db.Close()

	st := store.NewStore(db)
	found, err := st.LoadTxsWithMessages(ctx, []string{"escrow/create"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(found) == 0 {
		fmt.Println("no tx found with given message path")
	} else {
		fmt.Printf("messages found: %v", found)
	}
}
