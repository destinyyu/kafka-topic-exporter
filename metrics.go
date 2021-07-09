package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type serverConfig struct {
	port            int
	path            string
	expiration      int64
	exportTimestamp bool
}

func newServerConfig(port int, path string, expiration int64, exportTimestamp bool) serverConfig {
	if port < 0 || port > math.MaxUint16 {
		log.Fatal("Invalid port number")
	}
	return serverConfig{
		port:            port,
		path:            path,
		expiration:      expiration,
		exportTimestamp: exportTimestamp,
	}
}

func startMetricsServer(wg *sync.WaitGroup, shutdown chan struct{}, cfg serverConfig) {
	go func() {
		wg.Add(1)
		defer wg.Done()

		mux := http.NewServeMux()
		mux.Handle(cfg.path, promhttp.Handler())
		srv := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.port),
			Handler: mux,
		}

		go func() {
			log.WithField("port", cfg.port).
				WithField("path", cfg.path).
				Info("Starting metrics HTTP server")
			srv.ListenAndServe()
		}()

		<-shutdown
		log.Info("Shutting down metrics HTTP server")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
}

func generateMetrics(wg *sync.WaitGroup, shutdown chan struct{}, cfg serverConfig) {
	go func() {
		wg.Add(1)
		defer wg.Done()

		collector := NewMetricCollector(cfg.expiration, cfg.exportTimestamp)
		for {
			select {
			case bs := <-MsgChan:
				kv := make(map[string]interface{})
				if err := json.Unmarshal(bs, &kv); err == nil {
					buildMetric(collector, kv)
				}
			case <-shutdown:
				log.Info("Shutting down metrics generation")
				return
			}
		}
	}()
}

func buildMetric(collector *MetricCollector, kv map[string]interface{}) {
	for _, dl := range dropLabels {
		delete(kv, dl)
	}

	valueType := kv["valueType"].(string)
	delete(kv, "valueType")

	metric := kv["metric"].(string)
	delete(kv, "metric")

	metric, ok := validMetricName(metric)
	if !ok {
		log.WithField("metric", metric).Warn("unvalid metric name")
	}

	switch valueType {
	case "GAUGE":
		collector.AddMetric(metric, prometheus.GaugeValue, kv)
	case "COUNTER":
		collector.AddMetric(metric, prometheus.CounterValue, kv)
	default:
		log.WithField("metric", metric).WithField("valueType", valueType).Warn("unsupported value type")
		return
	}
}
