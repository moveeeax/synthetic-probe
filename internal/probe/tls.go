package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
)

func probeTLS(ctx context.Context, t config.Target) Result {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout.Std())
	defer cancel()

	host, _, err := net.SplitHostPort(t.Address)
	if err != nil {
		return Result{Err: err}
	}

	// We deliberately skip chain verification: the whole point is to report the
	// certificate's expiry even when it is self-signed or already expired. A
	// prober that refused to inspect a broken cert could never warn about it.
	dialer := &tls.Dialer{Config: &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, //nolint:gosec // expiry monitoring, not a client
	}}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", t.Address)
	dur := time.Since(start)
	if err != nil {
		return Result{Duration: dur, Err: err}
	}
	defer conn.Close()

	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return Result{Duration: dur, Err: errors.New("peer presented no certificate")}
	}
	leaf := state.PeerCertificates[0]
	return Result{
		Up:         true,
		Duration:   dur,
		CertExpiry: leaf.NotAfter,
	}
}
