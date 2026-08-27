package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	addr := flag.String("addr", envAddr(), "listen address")
	dataPath := flag.String("data", envOr("BOTDO_DATA", "botdo.json"), "path to JSON data file")
	apiKey := flag.String("api-key", os.Getenv("BOTDO_API_KEY"), "workspace API key (or BOTDO_API_KEY)")
	checkoutURL := flag.String("checkout-url", os.Getenv("BOTDO_CHECKOUT_URL"), "paid plan checkout URL (or BOTDO_CHECKOUT_URL)")

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
	router := api.NewRouter(s, ls, http.FS(sub), api.Config{
		APIKey:      *apiKey,
		CheckoutURL: *checkoutURL,
	})

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

func envAddr() string {
	if addr := strings.TrimSpace(os.Getenv("BOTDO_ADDR")); addr != "" {
		return addr
	}
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return ":" + port
	}
	return ":8080"
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
