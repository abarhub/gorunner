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

func Init(param config.Parametres, logger *logutils.Logger) {

	if param.Global.PrometheusActive {

		logger.Printf("Prometheus active")

		http.Handle("/metrics", promhttp.Handler())
		var url = ":2112"
		if len(param.Global.UrlMetrics) > 0 {
			url = param.Global.UrlMetrics
		}
		go func() {
			err := http.ListenAndServe(url, nil)
			if err != nil {
				logger.Printf("Erreur pour initialiser le serveur web: %v", err)
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
			},
			[]string{TACHE},
		)

		prometheus.MustRegister(taskSucces)
		prometheus.MustRegister(taskEchec)
		prometheus.MustRegister(taskDuration)

	} else {
		logger.Printf("Prometheus désactive")
	}
	prometheusActive = param.Global.PrometheusActive
}

func AjoutMetrics(nom string, succes bool, duree float64, logger *logutils.Logger) {

	if prometheusActive {
		if succes {
			taskSucces.WithLabelValues(nom).Inc()
		} else {
			taskEchec.WithLabelValues(nom).Inc()
		}

		taskDuration.WithLabelValues(nom).Observe(duree)
		logger.Printf("prometheus envoi duree %s: %v", nom, duree)
	}
}

func AjoutMetricsDuree(nom string, duree float64, logger *logutils.Logger) {
	if prometheusActive {
		taskDuration.WithLabelValues(nom).Observe(duree)
		logger.Printf("prometheus envoi duree %s: %v", nom, duree)
	}
}
