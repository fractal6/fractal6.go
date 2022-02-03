package handlers

import (
    "fractale/fractal6.go/db"
    "github.com/prometheus/client_golang/prometheus"
)


var (
	userCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "user_count",
		Help: "Current number of registered user.",
	})

	openTensionCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_tension_count",
		Help: "Current number of tensions.",
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

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(userCount)
	prometheus.MustRegister(openTensionCount)
	prometheus.MustRegister(circleCount)
	prometheus.MustRegister(labelCount)
}


func InstrumentationMeasures() {
	var count int

    count = db.GetDB().CountHas("User.username")
	userCount.Set(float64(count))

    count = db.GetDB().CountHas2("Tension.title", "Tension.status", "Open")
	openTensionCount.Set(float64(count))

    count = db.GetDB().CountHas("Node.name")
	circleCount.Set(float64(count))

    count = db.GetDB().CountHas("Label.name")
	labelCount.Set(float64(count))
}
