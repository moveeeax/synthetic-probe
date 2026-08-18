package runner

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/moveeeax/synthetic-probe/internal/config"
	"github.com/moveeeax/synthetic-probe/internal/metrics"
	"github.com/moveeeax/synthetic-probe/internal/probe"
)

func TestRunnerReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.yaml")
	writeConfig(t, path, "alpha")

	reg := prometheus.NewRegistry()
	r := New(path, metrics.New(reg), log.New(io.Discard, "", 0))

	var probed sync.Map
	r.probe = func(_ context.Context, tgt config.Target) probe.Result {
		probed.Store(tgt.Name, true)
		return probe.Result{Up: true, DNSRecords: 1}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		_ = r.Run(ctx, 20*time.Millisecond)
		close(done)
	}()

	waitFor(t, func() bool { _, ok := probed.Load("alpha"); return ok })

	// Advance mtime and swap the target; the reload loop should pick it up.
	time.Sleep(30 * time.Millisecond)
	writeConfig(t, path, "beta")
	waitFor(t, func() bool { _, ok := probed.Load("beta"); return ok })

	cancel()
	<-done
}

func writeConfig(t *testing.T, path, name string) {
	t.Helper()
	cfg := "default_interval: 20ms\ntargets:\n  - name: " + name + "\n    type: dns\n    host: localhost\n"
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
