package handlers

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "fractale/fractal6.go/db"
)


var (
	userCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "user_count",
		Help: "Current number of registered user.",
	})

	openTensionCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_tension_count",
		Help: "Current number of open tensions.",
	})

	closeTensionCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "close_tension_count",
		Help: "Current number of close tensions.",
	})

	circleCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "circle_count",
		Help: "Current number of circles.",
	})

	labelCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "label_count",
		Help: "Current number of labels.",
	})
)

func InstruHandler() http.Handler {
    // Create Handler from scratch
    r := prometheus.NewRegistry()

	// Metrics have to be registered to be exposed:
	r.MustRegister(userCount)
	r.MustRegister(openTensionCount)
	r.MustRegister(closeTensionCount)
	r.MustRegister(circleCount)
	r.MustRegister(labelCount)

    // More metrics
    //MustRegister(
    //    promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
    //    promcollectors.NewGoCollector(),
    //)

    return promhttp.HandlerFor(r, promhttp.HandlerOpts{})
}


func InstrumentationMeasures() {
	var count int

    count = db.GetDB().CountHas("User.username")
	userCount.Set(float64(count))

    count = db.GetDB().CountHas2("Tension.title", "Tension.status", "Open")
	openTensionCount.Set(float64(count))

    count = db.GetDB().CountHas2("Tension.title", "Tension.status", "Closed")
	closeTensionCount.Set(float64(count))

    count = db.GetDB().CountHas("Node.name")
	circleCount.Set(float64(count))

    count = db.GetDB().CountHas("Label.name")
	labelCount.Set(float64(count))
}

