package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/simonhege/varmomapo/pkg/drawers"
	"github.com/simonhege/varmomapo/pkg/drawers/debugdrawer"
	"github.com/simonhege/varmomapo/pkg/logging"
	tilehandler "github.com/simonhege/varmomapo/pkg/tileHandler"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type Config struct {
	Logging logging.Config
	Metrics bool
	Traces  bool
}

func main() {
	// Retrieve config
	cfg := Config{
		Logging: logging.Config{
			Handler: "json",
		},
		Metrics: false,
		Traces:  false,
	}

	// Setup logger
	logging.Setup(cfg.Logging)
	slog.Info("started")
	logBuildInfo()

	if cfg.Metrics {
		metricexporter, _ := stdoutmetric.New()

		mp := metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(metricexporter)),
			metric.WithResource(newResource()),
		)
		defer func() {
			if err := mp.Shutdown(context.Background()); err != nil {
				slog.Error("mp.Shutdown failed", "error", err)
				os.Exit(1)
			}
		}()
		otel.SetMeterProvider(mp)
	}

	if cfg.Traces {
		spanExporter, err := newExporter(os.Stdout)
		if err != nil {
			slog.Error("newExporter failed", "error", err)
			os.Exit(1)
		}
		tp := trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
			trace.WithBatcher(spanExporter),
			trace.WithResource(newResource()),
		)
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				slog.Error("tp.Shutdown failed", "error", err)
				os.Exit(1)
			}
		}()
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	}

	drawer := debugdrawer.New(256)
	ts := tilehandler.New(func(layerName string) drawers.Drawer {
		return drawer
	})

	wrappedHandler := otelhttp.NewHandler(ts, "tileHandler")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: wrappedHandler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			slog.Error("server shutdown failed", "error", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
	}
	<-idleConnsClosed

	slog.Info("stopped")
}

// newExporter returns a console exporter.
func newExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
	)
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNamespace("com.xdbsoft"),
			semconv.ServiceName("varmomapod"),
			semconv.ServiceVersion("v0.1.0"), //TODO use buidl info
			attribute.String("environment", "demo"),
		),
	)
	if err != nil {
		slog.Error("failed to create resource", "error", err)
	}
	return r
}

func logBuildInfo() {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		var settings []slog.Attr
		for _, setting := range buildInfo.Settings {
			settings = append(settings, slog.Attr{
				Key:   setting.Key,
				Value: slog.StringValue(setting.Value),
			})
		}
		slog.Info("build info", "go_version", buildInfo.GoVersion, "settings", settings)
	} else {
		slog.Warn("no build info available")
	}
}
