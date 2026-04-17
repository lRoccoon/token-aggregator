package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func filepathDir(p string) string { return filepath.Dir(p) }

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "listen address")
	dbPath := flag.String("db", envOr("DB_PATH", "/data/usage.db"), "sqlite path")
	token := flag.String("token", os.Getenv("INGEST_TOKEN"), "bearer token (empty disables auth)")
	publicURL := flag.String("public-url", os.Getenv("PUBLIC_URL"), "public URL baked into installer scripts (derived from request if empty)")
	tz := flag.String("timezone", envOr("TIMEZONE", "Asia/Shanghai"), "timezone used for 'today' boundary in /report")
	flag.Parse()

	store, err := OpenStore(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	dataDir := filepathDir(*dbPath)
	srv := &Server{
		Store:       store,
		Token:       *token,
		PublicURL:   *publicURL,
		Timezone:    *tz,
		MaxBodySize: 1 << 20,
		Prices:      NewPriceBook(dataDir),
	}

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if pusher := buildSlotPusher(store, *tz, dataDir); pusher != nil {
		go pusher.Run(ctx)
	}

	go func() {
		log.Printf("listening on %s db=%s tz=%s", *addr, *dbPath, *tz)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")
	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = httpServer.Shutdown(shCtx)
}

func buildSlotPusher(store *Store, tz, dataDir string) *SlotPusher {
	slotID := os.Getenv("LARK_SLOT_ID")
	credential := os.Getenv("LARK_SLOT_CREDENTIAL")
	if slotID == "" || credential == "" {
		log.Printf("slot pusher: disabled (LARK_SLOT_ID/LARK_SLOT_CREDENTIAL not set)")
		return nil
	}
	interval := 5 * time.Minute
	if v := os.Getenv("LARK_SLOT_INTERVAL"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("invalid LARK_SLOT_INTERVAL %q: %v", v, err)
		}
		if parsed < time.Second {
			log.Fatalf("LARK_SLOT_INTERVAL too small: %s", parsed)
		}
		interval = parsed
	}
	tmplPath := os.Getenv("LARK_SLOT_TEMPLATE_PATH")
	if tmplPath == "" {
		tmplPath = filepath.Join(dataDir, "slot_template.tmpl")
	}
	cfg := SlotConfig{
		SlotID:       slotID,
		Credential:   credential,
		APIURL:       envOr("LARK_SLOT_API_URL", defaultSlotAPIURL),
		Interval:     interval,
		TemplatePath: tmplPath,
		Timezone:     tz,
	}
	return NewSlotPusher(cfg, store)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
