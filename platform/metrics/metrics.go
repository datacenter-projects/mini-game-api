package metrics

import (
	"database/sql"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// namespace เดียวกับ v1 (askmelotto_minigame-api) เพื่อให้ dashboard/alert เดิมใช้ต่อได้
// ชื่อ metric และ label ของชุด HTTP ห้ามเปลี่ยน
const namespace = "mini_game_api"

// Registry คือที่ลงทะเบียน metric ทั้งหมดของระบบ (ไม่ใช้ prometheus.DefaultRegisterer)
// metric ธุรกิจของ module ให้ประกาศในไฟล์ใหม่ของ package นี้ แล้วลงทะเบียนกับ Registry
var Registry = prometheus.NewRegistry()

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

func init() {
	Registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// RegisterDB ลงทะเบียน stat ของ connection pool (go_sql_* label db_name) — key ของ map คือ db_name
func RegisterDB(pools map[string]*sql.DB) {
	for name, db := range pools {
		Registry.MustRegister(collectors.NewDBStatsCollector(db, name))
	}
}

// RegisterRedis ลงทะเบียน stat ของ connection pool ของ Redis
func RegisterRedis(client *redis.Client) {
	Registry.MustRegister(newRedisPoolCollector(client))
}

// Handler ตอบ /metrics ในรูปแบบ Prometheus
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{Registry: Registry})
}
