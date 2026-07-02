// Package reconciliation runs the ledger's reconciliation checks on a
// schedule and alerts (via structured logs and Prometheus metrics) when
// drift is detected, rather than relying solely on the on-demand endpoint.
package reconciliation

import (
	"context"
	"log/slog"
	"time"

	"github.com/iho/goledger/internal/infrastructure/metrics"
	"github.com/iho/goledger/internal/usecase"
)

// Reconciler is the subset of ReconciliationUseCase the scheduler depends
// on, so tests can supply a fake without a real database.
type Reconciler interface {
	GenerateReconciliationReport(ctx context.Context) (*usecase.ReconciliationReport, error)
}

// Scheduler periodically runs a full reconciliation report and alerts on drift.
type Scheduler struct {
	reconciliationUC Reconciler
	logger           *slog.Logger
	metrics          *metrics.Metrics
	interval         time.Duration
}

// Config for Scheduler.
type Config struct {
	ReconciliationUC Reconciler
	Logger           *slog.Logger
	Metrics          *metrics.Metrics
	Interval         time.Duration
}

// NewScheduler creates a new reconciliation Scheduler.
func NewScheduler(cfg Config) *Scheduler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Scheduler{
		reconciliationUC: cfg.ReconciliationUC,
		logger:           cfg.Logger,
		metrics:          cfg.Metrics,
		interval:         cfg.Interval,
	}
}

// Start runs reconciliation on a ticker until the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("reconciliation scheduler started", slog.Duration("interval", s.interval))

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("reconciliation scheduler shutting down")
			return ctx.Err()
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

// runOnce executes a single reconciliation pass, logging and alerting on
// drift. Errors running the check itself are logged but never fatal to the
// scheduler loop.
func (s *Scheduler) runOnce(ctx context.Context) {
	start := time.Now()

	report, err := s.reconciliationUC.GenerateReconciliationReport(ctx)

	duration := time.Since(start)
	if s.metrics != nil {
		s.metrics.ReconciliationDuration.Observe(duration.Seconds())
	}

	if err != nil {
		s.logger.Error("reconciliation run failed", slog.String("error", err.Error()))
		if s.metrics != nil {
			s.metrics.ReconciliationRuns.WithLabelValues("error").Inc()
		}
		return
	}

	if s.metrics != nil {
		s.metrics.ReconciliationDiscrepancies.Set(float64(len(report.Discrepancies)))
		s.metrics.ReconciliationChainBreaks.Set(float64(len(report.ChainBreaks)))
	}

	drifted := len(report.Discrepancies) > 0 || len(report.ChainBreaks) > 0 || !report.LedgerConsistent

	if !drifted {
		s.logger.Info("reconciliation run clean",
			slog.Int("total_accounts", report.TotalAccounts),
			slog.Duration("duration", duration))
		if s.metrics != nil {
			s.metrics.ReconciliationRuns.WithLabelValues("ok").Inc()
		}
		return
	}

	// Drift detected: this is the alert. In the absence of a paging
	// integration, an ERROR-level structured log plus the gauges above is
	// the alerting surface for whatever scrapes/ships these logs and metrics.
	for _, d := range report.Discrepancies {
		s.logger.Error("reconciliation drift: balance does not match entry sum",
			slog.String("account_id", d.AccountID),
			slog.String("recorded_balance", d.RecordedBalance.String()),
			slog.String("calculated_balance", d.CalculatedBalance.String()),
			slog.String("difference", d.Difference.String()))
	}

	for _, c := range report.ChainBreaks {
		for _, b := range c.Breaks {
			s.logger.Error("reconciliation drift: entry chain broken",
				slog.String("account_id", c.AccountID),
				slog.String("entry_id", b.EntryID),
				slog.Int("sequence", b.Sequence),
				slog.String("reason", b.Reason))
		}
	}

	if !report.LedgerConsistent {
		s.logger.Error("reconciliation drift: ledger inconsistent across currencies")
	}

	if s.metrics != nil {
		s.metrics.ReconciliationRuns.WithLabelValues("drift").Inc()
	}
}
