package probe

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
)

func probeHTTP(ctx context.Context, t config.Target) Result {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout.Std())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, t.Method, t.URL, nil)
	if err != nil {
		return Result{Err: err}
	}

	client := &http.Client{
		// Record the first response rather than chasing redirects, so
		// expect_status checks the endpoint's real status and not the target
		// of a 301/302.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	dur := time.Since(start)
	if err != nil {
		return Result{Duration: dur, Err: err}
	}
	defer resp.Body.Close()
	// Drain the body so the connection can be reused and the timing is honest.
	_, _ = io.Copy(io.Discard, resp.Body)

	res := Result{
		Duration:   dur,
		HTTPStatus: resp.StatusCode,
		Up:         resp.StatusCode == t.ExpectStatus,
	}
	// If the endpoint served TLS, capture the leaf certificate's expiry as a
	// bonus signal — no need to configure a separate tls check for HTTPS URLs.
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		res.CertExpiry = resp.TLS.PeerCertificates[0].NotAfter
	}
	return res
}
