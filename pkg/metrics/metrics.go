package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// http_requests_total - количество запросов
// http_response_time_seconds - время обработки запросов

// Глобальные метрики
var (
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_time_seconds",
			Help:    "HTTP response time distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader с защитой от двойной записи заголовков
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code // Сохраняем статус для метрик
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code) // Отправляем статус клиенту
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader { // Если статус ещё не установлен
		rw.WriteHeader(http.StatusOK) // Автоматически ставим 200 OK
	}
	return rw.ResponseWriter.Write(b) // Отправляем данные клиенту
}

// Middleware для сбора метрик
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Прокси для захвата статуса ответа
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK, // статус по умолчанию
		}

		next.ServeHTTP(rw, r)

		// Записываем метрики
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		status := strconv.Itoa(rw.status)

		httpRequests.WithLabelValues(r.Method, path, status).Inc()
		httpResponseTime.WithLabelValues(r.Method, path, status).Observe(duration)
	})
}

// Метрики на порту 9090
func RunMetricsServer() error {

	// сбор системных метрик
	startGoroutineMonitor(5 * time.Second)
	startMemoryMonitor(10 * time.Second)
	startGCFreedMemoryMonitor(15 * time.Second)
	startGCMonitor(15 * time.Second)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:    ":9090",
		Handler: metricsMux,
	}
	return metricsServer.ListenAndServe()
}

/*

################################
### Общее количество запросов
# Запросы по методам
sum by (method) (http_requests_total)

# Запросы по путям
sum by (path) (http_requests_total)

# Топ-5 самых популярных эндпоинтов
topk(5, sum by (path) (http_requests_total))

################################
### Rate-запросы (запросов в секунду):
# RPS (requests per second) всего
rate(http_requests_total[1m])

# RPS по методам
sum by (method) (rate(http_requests_total[1m]))

# RPS по конкретному пути
rate(http_requests_total{path="/your/endpoint"}[1m])

################################
### Базовые запросы:
# Общее количество наблюдений
http_response_time_seconds_count

# Общее суммарное время
http_response_time_seconds_sum

# Среднее время ответа
rate(http_response_time_seconds_sum[5m]) / rate(http_response_time_seconds_count[5m])
Перцентили:

################################
### Перцентили:
# 95-й перцентиль времени ответа
histogram_quantile(0.95, sum by (le, method, path) (rate(http_response_time_seconds_bucket[5m])))

# 50-й перцентиль (медиана)
histogram_quantile(0.5, sum by (le) (rate(http_response_time_seconds_bucket[5m])))

# 99-й перцентиль
histogram_quantile(0.99, sum by (le) (rate(http_response_time_seconds_bucket[5m])))

################################
### Группировки:
# Время ответа по методам
histogram_quantile(0.95, sum by (le, method) (rate(http_response_time_seconds_bucket[5m])))

# Время ответа по путям
histogram_quantile(0.95, sum by (le, path) (rate(http_response_time_seconds_bucket[5m])))

*/
