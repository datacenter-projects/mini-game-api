package metrics

import (
	"net/http"
	"time"
)

// NewServer สร้าง HTTP server ของ /metrics แยกจาก API (METRICS_ADDR) — พอร์ตนี้ใช้ภายใน cluster เท่านั้น
// ห้ามเปิดผ่าน ingress
func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", Handler())
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
