// Command synthetic-probe periodically checks HTTP, DNS, and TLS targets from a
// YAML config and exposes the results as Prometheus metrics.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/moveeeax/synthetic-probe/internal/metrics"
	"github.com/moveeeax/synthetic-probe/internal/runner"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cfgPath := flag.String("config", "targets.yaml", "path to the YAML target config")
	listen := flag.String("listen", ":9115", "address for the metrics HTTP server")
	reload := flag.Duration("reload-interval", 10*time.Second, "how often to check the config file for changes")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("synthetic-probe", version)
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	reg := prometheus.NewRegistry()
	m := metrics.New(reg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	r := runner.New(*cfgPath, m, logger)
	go func() {
		if err := r.Run(ctx, *reload); err != nil {
			logger.Fatalf("runner: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1>synthetic-probe</h1><a href="/metrics">/metrics</a></body></html>`))
	})

	srv := &http.Server{
		Addr:              *listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Printf("synthetic-probe %s listening on %s (config %s)", version, *listen, *cfgPath)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("http server: %v", err)
	}
}
