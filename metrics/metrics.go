package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	PERFORMANCE_LOG_DURATION = 60
)

var (
	Events = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sp_events",
			Help: ": number of events received from network",
		},
		[]string{"type"},
	)

	ConnNumbers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sp_conn_connection_numbers",
			Help: ": number of connections",
		},
		[]string{"type"})

	ConnReconnection = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sp_conn_reconnection_counters",
			Help: ": number of re-connections from each ip address",
		},
		[]string{"ip_address"},
	)

	InboundSpeed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pp_inbound_speed",
			Help: ": inbound speed from slice related traffic",
		},
		[]string{"opponent_p2p_address"})

	OutboundSpeed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pp_outbound_speed",
			Help: ": outbound speed from slice related traffic",
		},
		[]string{"opponent_p2p_address"})

	TaskCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pp_task_cnt",
			Help: ": count of tasks",
		},
		[]string{"task_cnt"})

	StoredSliceCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pp_stored_slices_cnt",
			Help: ": count of stored slices",
		},
		[]string{"stored_slices_cnt"})

	RpcReqCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pp_rpc_req_cnt",
			Help: ": count of rpc requests",
		},
		[]string{"rpc_req_cnt"})

	UploadProfiler = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "file_upload_profiler",
			Help: ": time for file upload",
		},
		[]string{"checkpoint"})

	DownloadProfiler = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "file_download_profiler",
			Help: ": time for file download",
		},
		[]string{"checkpoint"})

	IsLoggingPerformanceData bool

	LogPerformanceStartTime int64
)

func StartLoggingPerformanceData() {
	IsLoggingPerformanceData = true
	LogPerformanceStartTime = time.Now().Unix()
}

func UploadPerformanceLogNow(index string) {
	if time.Now().Unix()-LogPerformanceStartTime >= PERFORMANCE_LOG_DURATION {
		IsLoggingPerformanceData = false
	}

	if IsLoggingPerformanceData {
		UploadProfiler.WithLabelValues(index).Set(float64(time.Now().UnixMicro()))
	}
}

func UploadPerformanceLogData(index string, data int64) {
	if time.Now().Unix()-LogPerformanceStartTime >= PERFORMANCE_LOG_DURATION {
		IsLoggingPerformanceData = false
	}

	if IsLoggingPerformanceData {
		UploadProfiler.WithLabelValues(index).Set(float64(data))
	}
}

func DownloadPerformanceLogNow(index string) {
	if time.Now().Unix()-LogPerformanceStartTime >= PERFORMANCE_LOG_DURATION {
		IsLoggingPerformanceData = false
	}

	if IsLoggingPerformanceData {
		DownloadProfiler.WithLabelValues(index).Set(float64(time.Now().UnixMicro()))
	}
}
