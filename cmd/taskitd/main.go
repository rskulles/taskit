package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rskulles/taskit/pkg/api"
	"github.com/rskulles/taskit/pkg/store/sqlite"
)

func main() {
	var (
		addr = flag.String("addr", ":42069", "listen address")
		dsn  = flag.String("db", defaultDB(), "sqlite database path")
	)
	flag.Parse()

	store, err := sqlite.New(*dsn)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	srv := api.NewServer(store)
	fmt.Printf("taskitd listening on %s (db: %s)\n", *addr, *dsn)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatal(err)
	}
}

func defaultDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "taskit.db"
	}
	dir := filepath.Join(home, ".local", "share", "taskit")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "taskit.db")
}
