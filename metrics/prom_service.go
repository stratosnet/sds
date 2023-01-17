package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func Initialize(port string) error {
	// Metrics have to be registered to be exposed:
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":"+port, nil)
	return nil
}
