package config

import (
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	raw := []byte(`
default_interval: 30s
targets:
  - name: web
    type: http
    url: https://example.com
  - name: dns-check
    type: dns
    host: example.com
    interval: 1m
  - name: cert
    type: tls
    address: example.com:443
`)
	c, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(c.Targets) != 3 {
		t.Fatalf("want 3 targets, got %d", len(c.Targets))
	}

	web := c.Targets[0]
	if web.Method != "GET" {
		t.Errorf("http method should default to GET, got %q", web.Method)
	}
	if web.ExpectStatus != 200 {
		t.Errorf("expect_status should default to 200, got %d", web.ExpectStatus)
	}
	if web.Interval.Std() != 30*time.Second {
		t.Errorf("interval should fall back to default_interval, got %s", web.Interval.Std())
	}
	if c.Targets[1].Interval.Std() != time.Minute {
		t.Errorf("explicit interval not honored: %s", c.Targets[1].Interval.Std())
	}
	if c.Targets[2].Timeout.Std() != defaultTimeout {
		t.Errorf("timeout should default, got %s", c.Targets[2].Timeout.Std())
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"missing url":  "targets:\n  - name: x\n    type: http\n",
		"missing host": "targets:\n  - name: x\n    type: dns\n",
		"missing addr": "targets:\n  - name: x\n    type: tls\n",
		"unknown type": "targets:\n  - name: x\n    type: ping\n    url: http://x\n",
		"dup name":     "targets:\n  - name: x\n    type: dns\n    host: a\n  - name: x\n    type: dns\n    host: b\n",
		"no targets":   "default_interval: 5s\n",
		"bad duration": "targets:\n  - name: x\n    type: dns\n    host: a\n    interval: 5flurbs\n",
	}
	for name, raw := range cases {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}
