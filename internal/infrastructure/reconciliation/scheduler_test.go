package reconciliation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/infrastructure/metrics"
	"github.com/iho/goledger/internal/infrastructure/reconciliation"
	"github.com/iho/goledger/internal/usecase"
)

type fakeReconciler struct {
	report *usecase.ReconciliationReport
	err    error
	calls  int
}

func (f *fakeReconciler) GenerateReconciliationReport(ctx context.Context) (*usecase.ReconciliationReport, error) {
	f.calls++
	return f.report, f.err
}

// newTestMetrics registers metrics against a fresh registry, mirroring
// internal/infrastructure/metrics's own test, so each test in this package
// (which each construct their own metrics.Metrics) doesn't collide with the
// process-wide default Prometheus registry.
func newTestMetrics(t *testing.T) *metrics.Metrics {
	t.Helper()

	registry := prometheus.NewRegistry()
	prevRegisterer, prevGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = prevRegisterer, prevGatherer
	})

	return metrics.New()
}

func runOnceViaShortLoop(t *testing.T, s *reconciliation.Scheduler) {
	t.Helper()
	// Start ticks immediately once on entry, then again after the interval;
	// cancel shortly after the first tick to bound the test.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.Start(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduler_CleanRunRecordsOkMetric(t *testing.T) {
	fake := &fakeReconciler{
		report: &usecase.ReconciliationReport{
			TotalAccounts:      2,
			ReconciledAccounts: 2,
			LedgerConsistent:   true,
		},
	}

	m := newTestMetrics(t)
	s := reconciliation.NewScheduler(reconciliation.Config{
		ReconciliationUC: fake,
		Metrics:          m,
		Interval:         time.Hour,
	})

	runOnceViaShortLoop(t, s)

	if fake.calls == 0 {
		t.Fatal("expected reconciliation report to be generated at least once")
	}

	if got := testutil.ToFloat64(m.ReconciliationRuns.WithLabelValues("ok")); got == 0 {
		t.Fatalf("expected ok run counter to be incremented, got %v", got)
	}
}

func TestScheduler_DriftRunRecordsDriftMetric(t *testing.T) {
	fake := &fakeReconciler{
		report: &usecase.ReconciliationReport{
			TotalAccounts:    2,
			LedgerConsistent: true,
			Discrepancies: []*usecase.ReconciliationResult{
				{
					AccountID:         "acc-1",
					RecordedBalance:   decimal.NewFromInt(100),
					CalculatedBalance: decimal.NewFromInt(80),
					Difference:        decimal.NewFromInt(20),
				},
			},
		},
	}

	m := newTestMetrics(t)
	s := reconciliation.NewScheduler(reconciliation.Config{
		ReconciliationUC: fake,
		Metrics:          m,
		Interval:         time.Hour,
	})

	runOnceViaShortLoop(t, s)

	if got := testutil.ToFloat64(m.ReconciliationRuns.WithLabelValues("drift")); got == 0 {
		t.Fatalf("expected drift run counter to be incremented, got %v", got)
	}

	if got := testutil.ToFloat64(m.ReconciliationDiscrepancies); got != 1 {
		t.Fatalf("expected 1 discrepancy recorded, got %v", got)
	}
}

func TestScheduler_ErrorRunRecordsErrorMetric(t *testing.T) {
	fake := &fakeReconciler{err: errors.New("db down")}

	m := newTestMetrics(t)
	s := reconciliation.NewScheduler(reconciliation.Config{
		ReconciliationUC: fake,
		Metrics:          m,
		Interval:         time.Hour,
	})

	runOnceViaShortLoop(t, s)

	if got := testutil.ToFloat64(m.ReconciliationRuns.WithLabelValues("error")); got == 0 {
		t.Fatalf("expected error run counter to be incremented, got %v", got)
	}
}
