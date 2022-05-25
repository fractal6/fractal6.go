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

	orgaCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orga_count",
		Help: "Current number of organisations.",
	})

	circleCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "circle_count",
		Help: "Current number of sub-circles.",
	})

	roleCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "role_count",
		Help: "Current number of roles.",
	})

	archiveCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "archive_count",
		Help: "Current number of archive nodes.",
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
	r.MustRegister(orgaCount)
	r.MustRegister(circleCount)
	r.MustRegister(roleCount)
	r.MustRegister(archiveCount)
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
	var count1 int
	var count2 int

    count = db.GetDB().CountHas("User.username")
	userCount.Set(float64(count))

    count = db.GetDB().CountHas2("Tension.title", "Tension.status", "Open")
	openTensionCount.Set(float64(count))

    count = db.GetDB().CountHas2("Tension.title", "Tension.status", "Closed")
	closeTensionCount.Set(float64(count))

    // orga
    count = db.GetDB().Count2("Node.isRoot", "true" , "Node.type_", "Circle", "uid")
	orgaCount.Set(float64(count))

    // circle
    count = db.GetDB().Count2("Node.isRoot", "false" , "Node.type_", "Circle", "uid")
	circleCount.Set(float64(count))

    // role (coordinator or peer)
    count1 = db.GetDB().Count2("Node.role_type", "Coordinator" , "Node.type_", "Role", "uid")
    count2 = db.GetDB().Count2("Node.role_type", "Peer" , "Node.type_", "Role", "uid")
	roleCount.Set(float64(count1+count2))

    // archived node
    count = db.GetDB().Count2("Node.isArchived", "true" , "Node.isRoot", "false", "uid")
	archiveCount.Set(float64(count))

    count = db.GetDB().CountHas("Label.name")
	labelCount.Set(float64(count))
}

