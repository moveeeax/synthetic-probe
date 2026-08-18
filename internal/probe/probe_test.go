package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moveeeax/synthetic-probe/internal/config"
)

func dur(d time.Duration) config.Duration { return config.Duration(d) }

func TestProbeHTTPUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := Probe(context.Background(), config.Target{
		Name: "t", Type: "http", URL: srv.URL, Method: "GET",
		ExpectStatus: 200, Timeout: dur(5 * time.Second),
	})
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if !res.Up || res.HTTPStatus != 200 {
		t.Fatalf("want up with status 200, got up=%v status=%d", res.Up, res.HTTPStatus)
	}
}

func TestProbeHTTPStatusMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := Probe(context.Background(), config.Target{
		Name: "t", Type: "http", URL: srv.URL, Method: "GET",
		ExpectStatus: 200, Timeout: dur(5 * time.Second),
	})
	if res.Up {
		t.Fatal("target should be down when status != expect_status")
	}
	if res.HTTPStatus != 503 {
		t.Fatalf("want observed status 503, got %d", res.HTTPStatus)
	}
}

func TestProbeDNS(t *testing.T) {
	res := Probe(context.Background(), config.Target{
		Name: "t", Type: "dns", Host: "localhost", Timeout: dur(5 * time.Second),
	})
	if res.Err != nil {
		t.Fatalf("resolving localhost failed: %v", res.Err)
	}
	if !res.Up || res.DNSRecords < 1 {
		t.Fatalf("want at least one record for localhost, got up=%v n=%d", res.Up, res.DNSRecords)
	}
}

func TestProbeTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")

	res := Probe(context.Background(), config.Target{
		Name: "t", Type: "tls", Address: addr, Timeout: dur(5 * time.Second),
	})
	if res.Err != nil {
		t.Fatalf("tls probe failed: %v", res.Err)
	}
	if !res.Up || res.CertExpiry.IsZero() {
		t.Fatalf("want a captured cert expiry, got up=%v expiry=%v", res.Up, res.CertExpiry)
	}
	if days := res.CertDaysLeft(time.Now()); days <= 0 {
		t.Fatalf("httptest cert should have positive days left, got %v", days)
	}
}

func TestProbeUnknownType(t *testing.T) {
	res := Probe(context.Background(), config.Target{Name: "t", Type: "smoke"})
	if res.Err == nil {
		t.Fatal("want an error for an unknown probe type")
	}
}
