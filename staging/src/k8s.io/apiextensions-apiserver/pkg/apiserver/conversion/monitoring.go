/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	namespace = "apiserver"
	subsystem = "crd"
)

var (
	latencyBuckets = prometheus.ExponentialBuckets(1, 2, 5)
)

// converterMonitoringFactory holds metrics for all CRD converters
type converterMonitoringFactory struct {
	latencies map[string]*prometheus.HistogramVec
}

func newConverterMonitoringFactory() *converterMonitoringFactory {
	return &converterMonitoringFactory{map[string]*prometheus.HistogramVec{}}
}

var _ crConverterInterface = &converterMonitoring{}

type converterMonitoring struct {
	delegate  crConverterInterface
	latencies *prometheus.HistogramVec
	labels    []string
	crdName   string
}

func (c *converterMonitoringFactory) monitor(converterName string, crdName string, converter crConverterInterface) (crConverterInterface, error) {
	metric, exists := c.latencies[converterName]
	if !exists {
		metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      fmt.Sprintf("crd_%s_conversion_latencies_seconds", converterName),
				Help:      fmt.Sprintf("CRD %s conversion latency in seconds", converterName),
				Buckets:   latencyBuckets,
			},
			[]string{"crdName", "fromVersion", "toVersion", "succeed"})
		err := prometheus.Register(metric)
		if err != nil {
			return nil, err
		}
		c.latencies[converterName] = metric
	}
	return &converterMonitoring{latencies: metric, delegate: converter, crdName: crdName}, nil
}

func (m *converterMonitoring) Convert(in runtime.Object, targetGVK schema.GroupVersionKind) (runtime.Object, error) {
	start := time.Now()
	obj, err := m.delegate.Convert(in, targetGVK)
	fromVersion := in.GetObjectKind().GroupVersionKind().Version
	toVersion := targetGVK.Version

	// only record this observation if the version is different
	if fromVersion != toVersion {
		m.latencies.WithLabelValues(
			m.crdName, fromVersion, toVersion, strconv.FormatBool(err == nil)).Observe(time.Since(start).Seconds())
	}
	return obj, err
}
