package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// redisPoolCollector อ่าน PoolStats ของ go-redis ทุกครั้งที่ถูก scrape
type redisPoolCollector struct {
	client *redis.Client

	hits       *prometheus.Desc
	misses     *prometheus.Desc
	timeouts   *prometheus.Desc
	totalConns *prometheus.Desc
	idleConns  *prometheus.Desc
}

func newRedisPoolCollector(client *redis.Client) *redisPoolCollector {
	desc := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc(prometheus.BuildFQName(namespace, "redis_pool", name), help, nil, nil)
	}
	return &redisPoolCollector{
		client:     client,
		hits:       desc("hits_total", "Number of times a free connection was found in the pool."),
		misses:     desc("misses_total", "Number of times a free connection was not found in the pool."),
		timeouts:   desc("timeouts_total", "Number of times a wait timeout occurred."),
		totalConns: desc("total_conns", "Number of total connections in the pool."),
		idleConns:  desc("idle_conns", "Number of idle connections in the pool."),
	}
}

func (c *redisPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.hits
	ch <- c.misses
	ch <- c.timeouts
	ch <- c.totalConns
	ch <- c.idleConns
}

func (c *redisPoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.client.PoolStats()
	ch <- prometheus.MustNewConstMetric(c.hits, prometheus.CounterValue, float64(s.Hits))
	ch <- prometheus.MustNewConstMetric(c.misses, prometheus.CounterValue, float64(s.Misses))
	ch <- prometheus.MustNewConstMetric(c.timeouts, prometheus.CounterValue, float64(s.Timeouts))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns))
}
