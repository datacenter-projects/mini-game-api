package metrics

import "github.com/prometheus/client_golang/prometheus"

// metric ของ cronjob (docs/modules/currency_rate.md CJ-06) — label job = code ใน cronjobs
var (
	// CronjobRunsTotal นับรอบที่รันจริง แยกตามผล (SUCCESS / FAILED / SKIPPED) — job ที่ปิดอยู่ไม่นับ
	CronjobRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cronjob_runs_total",
			Help:      "Total number of cronjob runs by status.",
		},
		[]string{"job", "status"},
	)

	// CronjobLastSuccessTimestamp คือเวลา (unix วินาที) ที่รอบสำเร็จล่าสุด — ตั้ง alert เมื่อห่างจากปัจจุบันเกินไป
	CronjobLastSuccessTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cronjob_last_success_timestamp",
			Help:      "Unix time of the last successful cronjob run.",
		},
		[]string{"job"},
	)
)

func init() {
	Registry.MustRegister(CronjobRunsTotal, CronjobLastSuccessTimestamp)
}
