package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/aliencodez/botdo/internal/api"
	"github.com/aliencodez/botdo/internal/store"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataPath := flag.String("data", "botdo.json", "path to JSON data file")
	flag.Parse()

	s, err := store.NewJSONStore(*dataPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	// Strip the "web/" prefix so files are served from root.
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}
	router := api.NewRouter(s, http.FS(sub))

	log.Printf("botdo listening on %s  (data: %s)", *addr, *dataPath)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}
}
