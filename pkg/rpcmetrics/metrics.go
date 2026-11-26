package rpcmetrics

import (
	"context"
	"net/http"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDKey = "request_id"

type Metrics struct {
	registry *prometheus.Registry
	calls    *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(service string) *Metrics {
	registry := prometheus.NewRegistry()
	labels := []string{"method", "code"}
	calls := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "grpc_requests_total",
		Help:        "Количество завершённых gRPC-вызовов.",
		ConstLabels: prometheus.Labels{"service": service},
	}, labels)
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "grpc_request_duration_seconds",
		Help:        "Длительность gRPC-вызовов в секундах.",
		ConstLabels: prometheus.Labels{"service": service},
	}, labels)
	registry.MustRegister(calls, duration)
	return &Metrics{registry: registry, calls: calls, duration: duration}
}

func (m *Metrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

func (m *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	return mux
}

func (m *Metrics) Client(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if id := logger.RequestID(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, requestIDKey, id)
	}
	start := time.Now()
	err := invoke(ctx, method, req, reply, conn, opts...)
	m.record(ctx, method, time.Since(start), err)
	return err
}

func (m *Metrics) Server(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		requestLogger := log
		ids := metadata.ValueFromIncomingContext(ctx, requestIDKey)
		if len(ids) == 1 {
			if _, err := uuid.Parse(ids[0]); err == nil {
				ctx = logger.WithRequestID(ctx, ids[0])
				requestLogger = log.With().Str(requestIDKey, ids[0]).Logger()
			}
		}
		ctx = requestLogger.WithContext(ctx)
		start := time.Now()
		reply, err := next(ctx, req)
		m.record(ctx, info.FullMethod, time.Since(start), err)
		return reply, err
	}
}

func (m *Metrics) record(ctx context.Context, method string, elapsed time.Duration, err error) {
	code := status.Code(err).String()
	m.calls.WithLabelValues(method, code).Inc()
	m.duration.WithLabelValues(method, code).Observe(elapsed.Seconds())
	zerolog.Ctx(ctx).Info().
		Str("grpc_method", method).
		Str("grpc_code", code).
		Dur("duration_ms", elapsed).
		Msg("gRPC-вызов завершён")
}
