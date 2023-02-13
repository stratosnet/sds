package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stratosnet/sds/utils"
)

func Initialize(port string) error {
	// Metrics have to be registered to be exposed:
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(":"+port, nil)
		if err != nil {
			utils.ErrorLog(err)
		}
	}()
	return nil
}
