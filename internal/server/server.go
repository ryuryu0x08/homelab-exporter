package server

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/ryuryu0x08/homelab-exporter/internal/aggregate"
)

const (
	metricsPath = "/metrics"
	healthPath  = "/healthz"
)

// Gatherer produces one aggregated Prometheus scrape.
type Gatherer interface {
	Gather(ctx context.Context) ([]byte, aggregate.GatherStatus)
}

// New creates the exporter HTTP handler.
func New(gatherer Gatherer, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(metricsPath, metricsHandler(gatherer, logger))
	mux.HandleFunc(healthPath, healthHandler)
	return mux
}

func metricsHandler(gatherer Gatherer, logger *log.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.Header().Set("Allow", http.MethodGet)
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, status := gatherer.Gather(request.Context())
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if status == aggregate.GatherStatusUnavailable {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
		_, err := writer.Write(body)
		if err != nil {
			logger.Printf("server.metricsHandler write response failed: %v", err)
			return
		}
	}
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
