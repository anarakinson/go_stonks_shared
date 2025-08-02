package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// сорка метрик о соличестве активных горутин
	goroutinesCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_total",
			Help: "Current number of goroutines",
		},
	)

	// сборка метрик об использованной памяти
	memoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage by category",
		},
		[]string{"type"}, // ["heap", "stack", "alloc"]
	)

	// сборка метрик о сборке мусора
	gcCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gc_runs_total",
			Help: "Total GC runs",
		},
	)

	gcFreedMemory = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gc_freed_memory_bytes_total",
			Help: "Total bytes of memory freed by garbage collector",
		},
	)
)

// старт фонового сбора данных о количестве горутин
func startGoroutineMonitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			// Получаем текущее количество горутин
			goroutinesCount.Set(float64(runtime.NumGoroutine()))
		}
	}()
}

// обновление данных о памяти программы
func collectMemoryMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Основные категории памяти
	memoryUsage.WithLabelValues("heap").Set(float64(m.HeapAlloc))
	memoryUsage.WithLabelValues("stack").Set(float64(m.StackInuse))
	memoryUsage.WithLabelValues("alloc").Set(float64(m.Alloc))
	memoryUsage.WithLabelValues("sys").Set(float64(m.Sys))

	// Дополнительные метрики
	memoryUsage.WithLabelValues("objects").Set(float64(m.HeapObjects))
}

// старт фонового сбора данных об используемой памяти
func startMemoryMonitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			collectMemoryMetrics()
		}
	}()
}

// старт фонового сбора данных о GC
func startGCMonitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var lastNumGC uint32
		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			current := m.NumGC
			if current > lastNumGC {
				gcCount.Add(float64(current - lastNumGC))
				lastNumGC = current
			}
		}
	}()
}

// старт фонового сбора данных об освобожденной памяти
func startGCFreedMemoryMonitor(interval time.Duration) {
	// Инициализация
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	lastHeapInUse := m.HeapInuse

	// Запуск фонового мониторинга
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Если использование кучи уменьшилось - значит GC освободил память
			if m.HeapInuse < lastHeapInUse {
				freed := lastHeapInUse - m.HeapInuse
				gcFreedMemory.Add(float64(freed))
			}

			lastHeapInUse = m.HeapInuse
		}
	}()
}

/*


# Текущее количество горутин
goroutines_total

# Максимальное значение за последний час
max_over_time(goroutines_total[1h])

# Рост количества горутин
rate(goroutines_total[5m])


# Потребление памяти по типам
memory_usage_bytes

# Потребление heap-памяти в MB
memory_usage_bytes{type="heap"} / 1024 / 1024

# Общий объем памяти
sum(memory_usage_bytes) by (instance)

# Утечка памяти (рост за 1h)
rate(memory_usage_bytes{type="heap"}[1h]) > 0



*/
