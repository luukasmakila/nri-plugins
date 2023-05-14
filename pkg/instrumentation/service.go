// Copyright 2019-2020 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instrumentation

import (
	"sync"

	"github.com/containers/nri-plugins/pkg/http"
	"github.com/containers/nri-plugins/pkg/instrumentation/metrics"
	"github.com/containers/nri-plugins/pkg/instrumentation/tracing"
)

// service is the state of our instrumentation services: HTTP endpoint, trace/metrics exporters.
type service struct {
	sync.RWMutex              // we're RW-lockable
	http         *http.Server // HTTP server
}

// newService creates an instance of our instrumentation services.
func newService() *service {
	return &service{
		http: http.NewServer(),
	}
}

// Start starts instrumentation services.
func (s *service) Start() error {
	log.Info("starting instrumentation services...")

	s.Lock()
	defer s.Unlock()

	err := s.http.Start(opt.HTTPEndpoint)
	if err != nil {
		return instrumentationError("failed to start HTTP server: %v", err)
	}

	err = s.startTracing()
	if err != nil {
		return instrumentationError("failed to start tracing exporter: %v", err)
	}

	err = s.startMetrics()
	if err != nil {
		return instrumentationError("failed to start metrics exporter: %v", err)
	}

	return nil
}

func (s *service) startTracing() error {
	return tracing.Start(
		tracing.WithServiceName(ServiceName),
		tracing.WithCollectorEndpoint(opt.TracingCollector),
		tracing.WithSamplingRatio(opt.Sampling.Ratio()),
	)
}

func (s *service) startMetrics() error {
	return metrics.Start(
		GetHTTPMux(),
		metrics.WithExporterDisabled(!opt.PrometheusExport),
		metrics.WithServiceName(ServiceName),
		metrics.WithPeriod(opt.ReportPeriod),
	)
}

// Stop stops instrumentation services.
func (s *service) Stop() {
	s.Lock()
	defer s.Unlock()

	s.stopMetrics()
	s.stopTracing()
	s.http.Stop()
}

func (s *service) stopTracing() {
	tracing.Stop()
}

func (s *service) stopMetrics() {
	metrics.Stop()
}

// reconfigure reconfigures instrumentation services.
func (s *service) reconfigure() error {
	s.Lock()
	defer s.Unlock()

	err := s.http.Reconfigure(opt.HTTPEndpoint)
	if err != nil {
		return instrumentationError("failed to reconfigure HTTP server: %v", err)
	}

	s.stopTracing()
	err = s.startTracing()
	if err != nil {
		return instrumentationError("failed to reconfigure tracing exporter: %v")
	}

	s.stopMetrics()
	err = s.startMetrics()
	if err != nil {
		return instrumentationError("failed to reconfigure metrics exporter: %v")
	}

	return nil
}

// Restart restarts instrumentation services.
func (s *service) Restart() error {
	s.Stop()
	return s.Start()
}
