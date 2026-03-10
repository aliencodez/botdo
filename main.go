package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/aliencodez/botdo/internal/agent"
	"github.com/aliencodez/botdo/internal/api"
	"github.com/aliencodez/botdo/internal/dispatch"
	"github.com/aliencodez/botdo/internal/logstore"
	"github.com/aliencodez/botdo/internal/store"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataPath := flag.String("data", "botdo.json", "path to JSON data file")

	// TODO: workspace from the flag is redundant since each space will have it's own working directory
	workspace := flag.String("workspace", ".", "working dir for agent execution")
	logDir := flag.String("log-dir", "logs", "dir for per-task log files")
	claudeBin := flag.String("claude-bin", "claude", "path to claude binary")
	noAgent := flag.Bool("no-agent", false, "disable dispatcher (pure REST mode)")
	flag.Parse()

	s, err := store.NewJSONStore(*dataPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	ls, err := logstore.NewFileLogStore(*logDir)
	if err != nil {
		log.Fatalf("logstore: %v", err)
	}

	// Strip the "web/" prefix so files are served from root.
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}
	router := api.NewRouter(s, ls, http.FS(sub))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !*noAgent {
		cc := &agent.ClaudeCodeAdapter{ClaudeBin: *claudeBin}
		// TODO: Dispatcher doesn't need the workspace path, instead it should take it from the space's settings
		d := dispatch.New(s, ls, *workspace, 0, cc)
		go d.Start(ctx)
		log.Printf("botdo dispatcher started (workspace: %s, log-dir: %s)", *workspace, *logDir)
	} else {
		log.Printf("botdo dispatcher disabled (--no-agent)")
	}

	log.Printf("botdo listening on %s  (data: %s)", *addr, *dataPath)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}
}
