package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	MsgChan    = make(chan []byte, 100000)
	dropLabels = []string{}
)

func setDropLabels(labels []string) {
	dropLabels = labels
}

type Sample struct {
	Name        string
	LabelNames  []string
	LabelValues []string
	Value       float64
	ValueType   prometheus.ValueType
	Timestamp   int64
}

func splitLabelsAndValues(labels map[string]string) ([]string, []string) {
	num := len(labels)
	names := make([]string, 0, num)
	values := make([]string, 0, num)

	for k, _ := range labels {
		names = append(names, k)
	}

	sort.Strings(names)
	for _, k := range names {
		values = append(values, labels[k])
	}
	return names, values
}

type SampleFamily struct {
	Samples map[string]*Sample
}

func NewSampleFamily(sample *Sample) *SampleFamily {
	samples := make(map[string]*Sample)
	uid := UID(sample.Name, strings.Join(sample.LabelNames, ","), strings.Join(sample.LabelValues, ","))
	samples[uid] = sample

	return &SampleFamily{
		Samples: samples,
	}
}

func (sf *SampleFamily) AddSample(sample *Sample) {
	uid := UID(sample.Name, strings.Join(sample.LabelNames, ","), strings.Join(sample.LabelValues, ","))
	sf.Samples[uid] = sample
}

type MetricCollector struct {
	sync.Mutex
	desc            *prometheus.Desc
	fam             map[string]*SampleFamily
	expiration      int64
	exportTimestamp bool
}

func NewMetricCollector(expiration int64, exportTimestamp bool) *MetricCollector {
	collector := &MetricCollector{
		desc: prometheus.NewDesc(
			"custom_metrics",
			"kafaka exporter for custom metrics",
			nil,
			nil,
		),
		fam:             make(map[string]*SampleFamily),
		expiration:      expiration,
		exportTimestamp: exportTimestamp,
	}

	prometheus.MustRegister(collector)

	return collector
}

func (c *MetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *MetricCollector) Collect(ch chan<- prometheus.Metric) {
	c.Lock()
	defer c.Unlock()

	c.expire(c.expiration)

	for name, family := range c.fam {
		for _, sample := range family.Samples {
			desc := prometheus.NewDesc(name, name, sample.LabelNames, nil)

			metric, err := prometheus.NewConstMetric(desc, sample.ValueType, sample.Value, sample.LabelValues...)
			if err != nil {
				log.WithField("metric", name).WithField("labelNames", sample.LabelNames).WithField("labelValues", sample.LabelValues).Error(err.Error())
				continue
			}

			if c.exportTimestamp {
				metric = prometheus.NewMetricWithTimestamp(time.Unix(sample.Timestamp, 0), metric)
			}

			ch <- metric
		}
	}
}

func (c *MetricCollector) expire(expiration int64) {
	if expiration == 0 {
		return
	}

	now := time.Now().Unix()
	for name, family := range c.fam {
		for key, sample := range family.Samples {
			if now-sample.Timestamp < expiration {
				continue
			}

			delete(family.Samples, key)
			if len(family.Samples) == 0 {
				delete(c.fam, name)
			}
		}
	}
}

func (c *MetricCollector) AddMetric(metric string, valueType prometheus.ValueType, kv map[string]interface{}) {
	value := kv["value"].(float64)
	delete(kv, "value")

	timestamp := time.Now().Unix()
	if ts, ok := kv["timestamp"]; ok {
		timestamp = ts.(int64)
		delete(kv, "timestamp")
	}

	labels := make(map[string]string)
	for k, v := range kv {
		labels[k] = v.(string)
	}

	labelNames, labelValues := splitLabelsAndValues(labels)
	sample := &Sample{
		Name:        metric,
		LabelNames:  labelNames,
		LabelValues: labelValues,
		Value:       value,
		ValueType:   valueType,
		Timestamp:   timestamp,
	}

	c.Lock()
	defer c.Unlock()

	family := c.fam[metric]
	if family == nil {
		family = NewSampleFamily(sample)
		c.fam[metric] = family
	} else {
		family.AddSample(sample)
	}

}
