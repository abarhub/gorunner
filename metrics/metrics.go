package metrics

import (
	"gorunner/config"
	"gorunner/logutils"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const TACHE = "tache"

var taskSucces *prometheus.CounterVec
var taskEchec *prometheus.CounterVec
var taskDuration *prometheus.HistogramVec
var prometheusActive bool

func Init(param config.Parametres) {

	if param.Global.PrometheusActive {

		logutils.Printf("Prometheus active")

		http.Handle("/metrics", promhttp.Handler())
		var url = ":2112"
		if len(param.Global.UrlMetrics) > 0 {
			url = param.Global.UrlMetrics
		}
		go func() {
			err := http.ListenAndServe(url, nil)
			if err != nil {
				logutils.Printf("Erreur pour initialiser le serveur web: %v", err)
				return
			}
		}()

		taskSucces = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gorunner_succes",
				Help: "Nombre total de succes pour les taches go runner",
			},
			[]string{TACHE},
		)
		taskEchec = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gorunner_echec",
				Help: "Nombre total d'échecs pour les taches go runner",
			},
			[]string{TACHE},
		)

		taskDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gorunner_task_duration_seconds",
				Help:    "Durée des tâches longues",
				Buckets: param.Global.PrometheusBuckets,
				/*[]float64{
					60, 120, 300, 600, 900, 1200, 1500, 1800, 2400, 3000, 3600,
					// 1min, 2min, 5min, 10min, 15min, 20min, 25min, 30min, 40min, 50min, 1h
				},*/

			},
			[]string{TACHE},
		)

		prometheus.MustRegister(taskSucces)
		prometheus.MustRegister(taskEchec)
		prometheus.MustRegister(taskDuration)

	} else {
		logutils.Printf("Prometheus désactive")
	}
	prometheusActive = param.Global.PrometheusActive
}

func AjoutMetrics(nom string, succes bool, duree float64) {

	if prometheusActive {
		if succes {
			taskSucces.WithLabelValues(nom).Inc()
		} else {
			taskEchec.WithLabelValues(nom).Inc()
		}

		taskDuration.WithLabelValues(nom).Observe(duree)
		logutils.Printf("prometheus envoi duree %s: %v", nom, duree)
	}
}

func AjoutMetricsDuree(nom string, duree float64) {
	if prometheusActive {
		taskDuration.WithLabelValues(nom).Observe(duree)
		logutils.Printf("prometheus envoi duree %s: %v", nom, duree)
	}
}
