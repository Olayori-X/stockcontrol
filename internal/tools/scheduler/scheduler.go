package scheduler

import (
	"context"
	"time"

	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

const defaultOverdueCheckInterval = time.Hour

// StartOverdueInvoiceCheck runs MarkOverdueInvoices once immediately, then
// on a repeating interval, until ctx is cancelled. Intended to be launched
// once in its own goroutine from main():
//
//	go scheduler.StartOverdueInvoiceCheck(ctx)
//
// The interval is read from the settings table on every cycle (not just at
// startup), so an admin changing overdue_check_interval_minutes takes
// effect on the next tick without a redeploy or restart.
func StartOverdueInvoiceCheck(ctx context.Context) {
	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Scheduler: failed to connect to database, overdue invoice check will not run: ", err)
		return
	}

	runOnce := func() {
		ids, err := (*database).MarkOverdueInvoices()
		if err != nil {
			log.Error("Scheduler: failed to mark overdue invoices: ", err)
			return
		}

		if len(ids) > 0 {
			log.Infof("Scheduler: marked %d invoice(s) overdue: %v", len(ids), ids)
			for _, id := range ids {
				(*database).RecordAuditLog(nil, "invoice_marked_overdue", id, "Automatically flagged overdue by scheduled check")
			}
		}
	}

	nextInterval := func() time.Duration {
		minutesStr, err := (*database).GetSettingFloat("overdue_check_interval_minutes")
		if err != nil {
			log.Warn("Scheduler: could not read overdue_check_interval_minutes, using default: ", err)
			return defaultOverdueCheckInterval
		}
		if minutesStr <= 0 {
			log.Warn("Scheduler: overdue_check_interval_minutes is not positive, using default")
			return defaultOverdueCheckInterval
		}
		return time.Duration(minutesStr) * time.Minute
	}

	log.Info("Scheduler: overdue invoice check started")
	runOnce() // don't wait a full interval before the first check

	timer := time.NewTimer(nextInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Scheduler: overdue invoice check stopped")
			return
		case <-timer.C:
			runOnce()
			timer.Reset(nextInterval())
		}
	}
}
