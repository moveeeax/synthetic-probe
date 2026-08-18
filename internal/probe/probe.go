// Package probe runs one-shot HTTP, DNS, and TLS checks against a target and
// reports a structured result the metrics layer can turn into gauges.
package probe

import (
	"context"
	"fmt"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
)

// Result is the outcome of a single probe run.
type Result struct {
	Up         bool
	Duration   time.Duration
	HTTPStatus int
	DNSRecords int
	CertExpiry time.Time // zero when the check has no certificate
	Err        error
}

// CertDaysLeft returns the number of days until the probed certificate expires,
// relative to now. It returns 0 when there is no certificate.
func (r Result) CertDaysLeft(now time.Time) float64 {
	if r.CertExpiry.IsZero() {
		return 0
	}
	return r.CertExpiry.Sub(now).Hours() / 24
}

// Probe dispatches to the check matching the target's type.
func Probe(ctx context.Context, t config.Target) Result {
	switch t.Type {
	case "http":
		return probeHTTP(ctx, t)
	case "dns":
		return probeDNS(ctx, t)
	case "tls":
		return probeTLS(ctx, t)
	default:
		return Result{Err: fmt.Errorf("unknown probe type %q", t.Type)}
	}
}
