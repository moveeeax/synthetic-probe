package probe

import (
	"context"
	"net"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
)

func probeDNS(ctx context.Context, t config.Target) Result {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout.Std())
	defer cancel()

	var resolver net.Resolver
	start := time.Now()
	addrs, err := resolver.LookupHost(ctx, t.Host)
	dur := time.Since(start)
	if err != nil {
		return Result{Duration: dur, Err: err}
	}
	return Result{
		Up:         len(addrs) > 0,
		Duration:   dur,
		DNSRecords: len(addrs),
	}
}
