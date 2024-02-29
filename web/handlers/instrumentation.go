/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"

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
	count = db.GetDB().Count2("Node.isRoot", "true", "Node.type_", "Circle", "uid")
	orgaCount.Set(float64(count))

	// circle
	count = db.GetDB().Count2("Node.isRoot", "false", "Node.type_", "Circle", "uid")
	circleCount.Set(float64(count))

	// role (coordinator or peer)
	count1 = db.GetDB().Count2("Node.role_type", "Coordinator", "Node.type_", "Role", "uid")
	count2 = db.GetDB().Count2("Node.role_type", "Peer", "Node.type_", "Role", "uid")
	roleCount.Set(float64(count1 + count2))

	// archived node
	count = db.GetDB().Count2("Node.isArchived", "true", "Node.isRoot", "false", "uid")
	archiveCount.Set(float64(count))

	count = db.GetDB().CountHas("Label.name")
	labelCount.Set(float64(count))
}
