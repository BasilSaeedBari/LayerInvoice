package jobs

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// Scheduler coordinates all automated temporal executions.
type Scheduler struct {
	db *sql.DB
	gs gocron.Scheduler
}

// NewScheduler instantiates the automated task manager.
func NewScheduler(db *sql.DB) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &Scheduler{db: db, gs: s}, nil
}

// Start launches background routines.
func (s *Scheduler) Start() error {
	// 1. Run CheckOverdue daily at 00:05 AM
	_, err := s.gs.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 5, 0))),
		gocron.NewTask(s.CheckOverdueInvoices),
	)
	if err != nil {
		return err
	}

	s.gs.Start()

	// Proactively run once on server boot
	go s.CheckOverdueInvoices()

	return nil
}

// Stop halts all active cron schedules.
func (s *Scheduler) Stop() error {
	return s.gs.Shutdown()
}

// CheckOverdueInvoices transitions invoices with past-due dates to overdue.
func (s *Scheduler) CheckOverdueInvoices() {
	slog.Info("Running automated check for overdue invoices...")

	today := time.Now().Format("2006-01-02")
	now := time.Now().Format(time.RFC3339)

	query := `
		UPDATE invoices 
		SET status = 'overdue', updated_at = ?
		WHERE status IN ('draft', 'pending', 'sent', 'partially_paid') 
		  AND due_date < ?
	`

	res, err := s.db.ExecContext(context.Background(), query, now, today)
	if err != nil {
		slog.Error("Failed to auto-update overdue invoices", "error", err)
		return
	}

	affected, _ := res.RowsAffected()
	slog.Info("Completed overdue invoice check", "invoices_marked_overdue", affected)
}
