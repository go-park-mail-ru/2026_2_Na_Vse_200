package middleware

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPMetrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewHTTPMetrics(service string) *HTTPMetrics {
	labels := prometheus.Labels{"service": service}
	metrics := &HTTPMetrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "Completed HTTP requests.", ConstLabels: labels,
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request duration in seconds.", ConstLabels: labels,
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.4, 0.5, 0.8, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
	}
	metrics.registry.MustRegister(metrics.requests, metrics.duration)
	return metrics
}

func (m *HTTPMetrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

func (m *HTTPMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

type metricsRouteKey struct{}

// Wrap измеряет запросы, включая ответы middleware до входа в ServeMux.
func (m *HTTPMetrics) Wrap(router *http.ServeMux, middlewares ...Middleware) http.Handler {
	next := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := r.Context().Value(metricsRouteKey{}).(*string)
		defer func() { *pattern = r.Pattern }()
		router.ServeHTTP(w, r)
	}), middlewares...)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var pattern string
		writer := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r.WithContext(context.WithValue(r.Context(), metricsRouteKey{}, &pattern)))
		duration := time.Since(started).Seconds()
		if r.URL.Path == "/metrics" {
			return
		}
		if pattern == "" {
			pattern = findRoutePattern(router, r, writer.Header().Get("Allow"))
		}
		_, route, hasMethod := strings.Cut(pattern, " ")
		if !hasMethod {
			route = pattern
		}
		if route == "" {
			route = "unmatched"
		}
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		method := boundedMethod(r.Method)
		m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
		m.duration.WithLabelValues(method, route).Observe(duration)
	})
}

var standardMethods = [...]string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
	http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
}

func findRoutePattern(router *http.ServeMux, r *http.Request, allow string) string {
	if _, pattern := router.Handler(r); pattern != "" {
		return pattern
	}

	probe := *r
	if r.Method == http.MethodOptions {
		probe.Method = r.Header.Get("Access-Control-Request-Method")
		if probe.Method != "" && probe.Method != http.MethodConnect {
			if _, pattern := router.Handler(&probe); pattern != "" {
				return pattern
			}
		}
	}

	methods := standardMethods[:]
	if allow != "" {
		methods = strings.Split(allow, ",")
	}
	for _, method := range methods {
		probe.Method = strings.TrimSpace(method)
		// Для CONNECT ServeMux может вернуть конкретный путь редиректа.
		if probe.Method == r.Method || probe.Method == http.MethodConnect {
			continue
		}
		if _, pattern := router.Handler(&probe); pattern != "" {
			return pattern
		}
	}
	return ""
}

func boundedMethod(method string) string {
	if slices.Contains(standardMethods[:], method) {
		return method
	}
	return "OTHER"
}
