package ingestion

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Router struct {
	mux            *http.ServeMux
	logger         *slog.Logger
	webhookHandler *WebhookHandler
}

func NewRouter(logger *slog.Logger) *Router {
	r := &Router{
		mux:            http.NewServeMux(),
		logger:         logger,
		webhookHandler: NewWebhookHandler(logger),
	}
	r.routes()
	return r
}

func (r *Router) routes() {
	r.mux.HandleFunc("/healthz", r.handleHealthz)
	r.mux.HandleFunc("/webhook", r.webhookHandler.HandleWebhook)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) handleHealthz(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "lucid-platform",
	})
}
