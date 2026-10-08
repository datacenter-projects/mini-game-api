package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestServerServesMetrics(t *testing.T) {
	HTTPRequestsTotal.WithLabelValues("GET", "/x", "200").Inc()
	srv := httptest.NewServer(NewServer(":0").Handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	for _, want := range []string{
		`mini_game_api_http_requests_total{method="GET",path="/x",status="200"}`,
		"go_goroutines",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in /metrics", want)
		}
	}

	// server นี้ตอบแค่ /metrics
	resp, err = http.Get(srv.URL + "/api/v1")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other path status = %d, want 404", resp.StatusCode)
	}
}

func TestRedisPoolCollector(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(t.Context()).Err(); err != nil {
		t.Fatal(err)
	}

	RegisterRedis(client)
	defer Registry.Unregister(newRedisPoolCollector(client))

	families, err := Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, f := range families {
		if strings.HasPrefix(f.GetName(), "mini_game_api_redis_pool_") {
			m := f.GetMetric()[0]
			got[f.GetName()] = m.GetCounter().GetValue() + m.GetGauge().GetValue()
		}
	}
	if len(got) != 5 {
		t.Fatalf("redis pool metrics = %v, want 5 series", got)
	}
	if got["mini_game_api_redis_pool_total_conns"] < 1 {
		t.Errorf("total_conns = %v, want >= 1 after ping", got["mini_game_api_redis_pool_total_conns"])
	}
}
