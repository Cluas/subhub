// Package scheduler wraps robfig/cron to auto-refresh subscriptions on their
// configured cron schedule.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/Cluas/subhub/internal/engine"
	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// Scheduler manages per-subscription cron jobs.
type Scheduler struct {
	c    *cron.Cron
	st   store.Store
	mu   sync.Mutex
	jobs map[int64]cron.EntryID // subscription ID → cron entry
}

// New creates a new Scheduler backed by the given store.
// Use cron.New() without WithSeconds — standard 5-field cron expressions.
func New(st store.Store) *Scheduler {
	return &Scheduler{
		c:    cron.New(),
		st:   st,
		jobs: make(map[int64]cron.EntryID),
	}
}

// Start begins the underlying cron scheduler.
func (s *Scheduler) Start() {
	s.c.Start()
}

// Stop gracefully stops the scheduler, waiting up to 10 s for in-flight jobs to
// finish before returning.
func (s *Scheduler) Stop() {
	ctx := s.c.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		slog.Warn("scheduler stop timed out after 10s; in-flight jobs may still be running")
	}
}

// Sync reconciles the set of active cron entries with the provided subscriptions.
//
//   - Entries for subs with AutoRefresh=false or empty RefreshCron are removed.
//   - Entries for subs with AutoRefresh=true and a non-empty RefreshCron are
//     added (or replaced when the spec changed).
//   - Subs that were already registered with the same spec are left unchanged.
func (s *Scheduler) Sync(ctx context.Context, subs []*model.Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build a quick-lookup set of active subscription IDs.
	active := make(map[int64]*model.Subscription, len(subs))
	for _, sub := range subs {
		active[sub.ID] = sub
	}

	// Remove entries for subscriptions that are no longer schedulable.
	for id, entryID := range s.jobs {
		sub, found := active[id]
		if !found || !sub.AutoRefresh || sub.RefreshCron == "" {
			s.c.Remove(entryID)
			delete(s.jobs, id)
			slog.Info("cron entry removed", "subscription_id", id)
		}
	}

	// Add or replace entries for schedulable subscriptions.
	for _, sub := range subs {
		if !sub.AutoRefresh || sub.RefreshCron == "" {
			continue
		}

		subID := sub.ID
		spec := sub.RefreshCron

		// Remove the old entry so we can replace it with the (possibly updated) spec.
		if oldID, exists := s.jobs[subID]; exists {
			s.c.Remove(oldID)
			delete(s.jobs, subID)
		}

		entryID, err := s.c.AddFunc(spec, func() {
			s.runRefresh(subID)
		})
		if err != nil {
			slog.Error("cron AddFunc failed", "subscription_id", subID, "spec", spec, "err", err)
			continue
		}

		s.jobs[subID] = entryID
		slog.Info("cron entry registered", "subscription_id", subID, "spec", spec)
	}
}

// runRefresh is the function executed by the cron job for a single subscription.
// It re-reads the subscription from the store to pick up any URL/config changes
// made since Sync was last called.
func (s *Scheduler) runRefresh(subID int64) {
	slog.Info("cron refresh triggered", "subscription_id", subID)

	ctx := context.Background()
	sub, err := s.st.GetSubscription(ctx, subID)
	if err != nil {
		slog.Error("cron refresh: failed to load subscription", "subscription_id", subID, "err", err)
		return
	}
	if sub == nil {
		slog.Warn("cron refresh: subscription not found, skipping", "subscription_id", subID)
		return
	}
	if !sub.AutoRefresh {
		// Sub was disabled between Sync and tick — skip silently.
		return
	}

	if err := engine.FetchAndPersist(ctx, s.st, sub); err != nil {
		slog.Error("cron refresh: fetch failed", "subscription_id", subID, "err", err)
	}
}
