package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/moveeeax/synthetic-probe/internal/config"
	"github.com/moveeeax/synthetic-probe/internal/probe"
)

func TestRecordHTTP(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	m.Record(config.Target{Name: "web", Type: "http"}, probe.Result{
		Up: true, Duration: 150 * time.Millisecond, HTTPStatus: 200,
	})

	if got := gaugeValue(t, reg, "synthetic_probe_up", "web"); got != 1 {
		t.Errorf("up = %v, want 1", got)
	}
	if got := gaugeValue(t, reg, "synthetic_probe_http_status_code", "web"); got != 200 {
		t.Errorf("status = %v, want 200", got)
	}
	if got := gaugeValue(t, reg, "synthetic_probe_duration_seconds", "web"); got != 0.15 {
		t.Errorf("duration = %v, want 0.15", got)
	}
}

func TestRecordTLSDays(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }

	m.Record(config.Target{Name: "cert", Type: "tls"}, probe.Result{
		Up: true, CertExpiry: now.Add(10 * 24 * time.Hour),
	})

	if got := gaugeValue(t, reg, "synthetic_probe_ssl_cert_expiry_days", "cert"); got < 9.99 || got > 10.01 {
		t.Errorf("cert days = %v, want ~10", got)
	}
}

func TestResetClearsStaleSeries(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	m.Record(config.Target{Name: "gone", Type: "dns"}, probe.Result{Up: true, DNSRecords: 2})
	m.Reset()

	mfs, _ := reg.Gather()
	for _, mf := range mfs {
		if len(mf.GetMetric()) != 0 {
			t.Errorf("metric %s should be empty after Reset, has %d series", mf.GetName(), len(mf.GetMetric()))
		}
	}
}

func gaugeValue(t *testing.T, reg *prometheus.Registry, name, target string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, mt := range mf.GetMetric() {
			if labelValue(mt, "target") == target {
				return mt.GetGauge().GetValue()
			}
		}
	}
	t.Fatalf("metric %s{target=%q} not found", name, target)
	return 0
}

func labelValue(m *dto.Metric, key string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == key {
			return l.GetValue()
		}
	}
	return ""
}
