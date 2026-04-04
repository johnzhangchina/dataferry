package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/johnzhangchina/dataferry/internal/handler"
	"github.com/johnzhangchina/dataferry/internal/store"
	"github.com/johnzhangchina/dataferry/web"
)

// Set via -ldflags at build time.
var version = "dev"

func main() {
	port := flag.Int("port", 8080, "server port")
	dbPath := flag.String("db", "dataferry.db", "SQLite database path")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("dataferry %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	// Override with env vars if set
	if p := os.Getenv("DATAFERRY_PORT"); p != "" {
		fmt.Sscanf(p, "%d", port)
	}
	if d := os.Getenv("DATAFERRY_DB"); d != "" {
		*dbPath = d
	}

	password := os.Getenv("DATAFERRY_PASSWORD")

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	auth := handler.NewAuthMiddleware(password)
	flowHandler := handler.NewFlowHandler(s)
	webhookHandler := handler.NewWebhookHandler(s)

	mux := http.NewServeMux()

	// Public endpoints (no auth)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		dbStatus := "ok"
		if err := s.Ping(); err != nil {
			status = "degraded"
			dbStatus = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  status,
			"version": version,
			"db":      dbStatus,
		})
	})
	mux.HandleFunc("POST /webhook/{path}", webhookHandler.Handle)
	mux.HandleFunc("POST /api/login", auth.Login)
	mux.HandleFunc("POST /api/logout", auth.Logout)
	mux.HandleFunc("GET /api/auth", auth.CheckAuth)

	// Protected management API
	mux.HandleFunc("GET /api/flows", auth.ProtectFunc(flowHandler.ListFlows))
	mux.HandleFunc("POST /api/flows", auth.ProtectFunc(flowHandler.CreateFlow))
	mux.HandleFunc("GET /api/flows/{id}", auth.ProtectFunc(flowHandler.GetFlow))
	mux.HandleFunc("PUT /api/flows/{id}", auth.ProtectFunc(flowHandler.UpdateFlow))
	mux.HandleFunc("DELETE /api/flows/{id}", auth.ProtectFunc(flowHandler.DeleteFlow))
	mux.HandleFunc("GET /api/flows/{id}/logs", auth.ProtectFunc(flowHandler.ListLogs))
	mux.HandleFunc("POST /api/flows/{id}/logs/{logId}/retry", auth.ProtectFunc(webhookHandler.Retry))

	// Frontend (SPA)
	mux.Handle("/", web.StaticHandler())

	addr := fmt.Sprintf(":%d", *port)
	if auth.Enabled() {
		log.Printf("DataFerry %s starting on %s (password protection enabled)", version, addr)
	} else {
		log.Printf("DataFerry %s starting on %s (no password set, management API is open)", version, addr)
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
