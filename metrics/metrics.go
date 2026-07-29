package metrics

import (
	"gorunner/config"
	"gorunner/logutils"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var taskDuration prometheus.Histogram

func Init(param config.Parametres) {
	http.Handle("/metrics", promhttp.Handler())
	var url = ":2112"
	if len(param.Global.UrlMetrics) > 0 {
		url = param.Global.UrlMetrics
	}
	err := http.ListenAndServe(url, nil)
	if err != nil {
		logutils.Printf("Erreur pour initialiser le serveur web: %v", err)
		return
	}

	taskDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gorunner_task_duration_seconds",
			Help: "Durée des tâches longues",
			Buckets: []float64{
				60, 120, 300, 600, 900, 1200, 1500, 1800, 2400, 3000, 3600,
				// 1min, 2min, 5min, 10min, 15min, 20min, 25min, 30min, 40min, 50min, 1h
			},
		},
	)

}
