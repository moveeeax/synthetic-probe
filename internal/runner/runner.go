// Package runner schedules the configured probes on their own intervals and
// hot-reloads the config file when it changes on disk.
package runner

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
	"github.com/moveeeax/synthetic-probe/internal/metrics"
	"github.com/moveeeax/synthetic-probe/internal/probe"
)

type probeFunc func(context.Context, config.Target) probe.Result

// Runner owns the per-target goroutines and the reload loop.
type Runner struct {
	path    string
	metrics *metrics.Metrics
	logger  *log.Logger
	probe   probeFunc // swappable in tests

	mu      sync.Mutex
	cancels []context.CancelFunc
	wg      sync.WaitGroup
}

// New builds a Runner that reads targets from path.
func New(path string, m *metrics.Metrics, logger *log.Logger) *Runner {
	return &Runner{path: path, metrics: m, logger: logger, probe: probe.Probe}
}

// Run loads the config, starts probing, and re-reads the file every
// reloadInterval, applying changes. It blocks until ctx is cancelled.
func (r *Runner) Run(ctx context.Context, reloadInterval time.Duration) error {
	cfg, err := config.Load(r.path)
	if err != nil {
		return err
	}
	r.apply(ctx, cfg)
	lastMod := modTime(r.path)

	ticker := time.NewTicker(reloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.stopAll()
			return nil
		case <-ticker.C:
			m := modTime(r.path)
			if m.Equal(lastMod) {
				continue
			}
			lastMod = m
			newCfg, err := config.Load(r.path)
			if err != nil {
				// Keep the previous config running rather than dropping all
				// probes because of a typo in the edit.
				r.logger.Printf("reload failed, keeping previous config: %v", err)
				continue
			}
			r.logger.Printf("reloaded %s: %d target(s)", r.path, len(newCfg.Targets))
			r.apply(ctx, newCfg)
		}
	}
}

// apply stops the current probe set and starts a fresh one for cfg.
func (r *Runner) apply(parent context.Context, cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cancel := range r.cancels {
		cancel()
	}
	r.cancels = nil
	r.wg.Wait() // ensure old goroutines exit before we reset/replace metrics
	r.metrics.Reset()

	for _, t := range cfg.Targets {
		ctx, cancel := context.WithCancel(parent)
		r.cancels = append(r.cancels, cancel)
		r.wg.Add(1)
		go r.runTarget(ctx, t)
	}
}

func (r *Runner) stopAll() {
	r.mu.Lock()
	for _, cancel := range r.cancels {
		cancel()
	}
	r.cancels = nil
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *Runner) runTarget(ctx context.Context, t config.Target) {
	defer r.wg.Done()

	// Probe once up front so /metrics is populated before the first interval.
	r.once(ctx, t)

	ticker := time.NewTicker(t.Interval.Std())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.once(ctx, t)
		}
	}
}

func (r *Runner) once(ctx context.Context, t config.Target) {
	res := r.probe(ctx, t)
	r.metrics.Record(t, res)
	if res.Err != nil {
		r.logger.Printf("probe %s (%s): %v", t.Name, t.Type, res.Err)
	}
}

func modTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
