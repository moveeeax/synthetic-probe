// Package metrics maps probe results onto Prometheus gauges and exposes them
// through a registry the HTTP handler serves at /metrics.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/moveeeax/synthetic-probe/internal/config"
	"github.com/moveeeax/synthetic-probe/internal/probe"
)

const namespace = "synthetic_probe"

// Metrics holds the gauges the prober exports.
type Metrics struct {
	up         *prometheus.GaugeVec
	duration   *prometheus.GaugeVec
	httpStatus *prometheus.GaugeVec
	dnsRecords *prometheus.GaugeVec
	certDays   *prometheus.GaugeVec
	certExpiry *prometheus.GaugeVec

	// now is overridable in tests so cert-day math is deterministic.
	now func() time.Time
}

// New builds the metric vectors and registers them.
func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		up: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "up",
			Help: "1 if the target passed its last check, 0 otherwise.",
		}, []string{"target", "type"}),
		duration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "duration_seconds",
			Help: "Duration of the last probe in seconds.",
		}, []string{"target", "type"}),
		httpStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "http_status_code",
			Help: "HTTP status code observed by the last http probe.",
		}, []string{"target"}),
		dnsRecords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "dns_records",
			Help: "Number of addresses returned by the last dns lookup.",
		}, []string{"target"}),
		certDays: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "ssl_cert_expiry_days",
			Help: "Days until the probed TLS certificate expires.",
		}, []string{"target"}),
		certExpiry: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "ssl_cert_expiry_timestamp_seconds",
			Help: "Unix timestamp at which the probed TLS certificate expires.",
		}, []string{"target"}),
		now: time.Now,
	}
	reg.MustRegister(m.up, m.duration, m.httpStatus, m.dnsRecords, m.certDays, m.certExpiry)
	return m
}

// Record updates the metric set for one target from a probe result.
func (m *Metrics) Record(t config.Target, r probe.Result) {
	up := 0.0
	if r.Up {
		up = 1
	}
	m.up.WithLabelValues(t.Name, t.Type).Set(up)
	m.duration.WithLabelValues(t.Name, t.Type).Set(r.Duration.Seconds())

	switch t.Type {
	case "http":
		m.httpStatus.WithLabelValues(t.Name).Set(float64(r.HTTPStatus))
	case "dns":
		m.dnsRecords.WithLabelValues(t.Name).Set(float64(r.DNSRecords))
	}

	if !r.CertExpiry.IsZero() {
		m.certDays.WithLabelValues(t.Name).Set(r.CertDaysLeft(m.now()))
		m.certExpiry.WithLabelValues(t.Name).Set(float64(r.CertExpiry.Unix()))
	}
}

// Reset drops all series. Called on config reload so metrics for targets that
// were removed do not linger as stale samples.
func (m *Metrics) Reset() {
	m.up.Reset()
	m.duration.Reset()
	m.httpStatus.Reset()
	m.dnsRecords.Reset()
	m.certDays.Reset()
	m.certExpiry.Reset()
}
