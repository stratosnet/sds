package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
)
