package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

type Config struct {
	ServiceName string
	Port        int
}

type Provider struct {
	provider *metric.MeterProvider
	server   *http.Server
}

func Setup(cfg Config) (*Provider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() { _ = srv.ListenAndServe() }()

	return &Provider{provider: provider, server: srv}, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	_ = p.server.Shutdown(ctx)
	return p.provider.Shutdown(ctx)
}
