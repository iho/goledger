package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRegistersMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()

	// Replace global default registry to allow test inspection.
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry

	m := New()

	if m.TransfersCreated == nil || m.HTTPRequests == nil || m.DBQueries == nil {
		t.Fatalf("expected key metrics to be initialized: %+v", m)
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Fatalf("expected registered metrics, got none")
	}
}
