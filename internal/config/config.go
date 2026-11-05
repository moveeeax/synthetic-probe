// Package config loads and validates the YAML target file that drives the
// prober. It applies sensible defaults so a minimal config still works.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	defaultInterval = 15 * time.Second
	defaultTimeout  = 5 * time.Second
)

// Duration is a time.Duration that unmarshals from a Go duration string such
// as "15s" or "1h", instead of a raw nanosecond integer.
type Duration time.Duration

// UnmarshalYAML parses a duration string like "30s".
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Std returns the value as a standard time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Target is a single thing to probe.
type Target struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"` // http, dns, or tls
	URL          string   `yaml:"url"`
	Host         string   `yaml:"host"`
	Address      string   `yaml:"address"`
	Method       string   `yaml:"method"`
	ExpectStatus int      `yaml:"expect_status"`
	Interval     Duration `yaml:"interval"`
	Timeout      Duration `yaml:"timeout"`
}

// Config is the top-level target file.
type Config struct {
	DefaultInterval Duration `yaml:"default_interval"`
	DefaultTimeout  Duration `yaml:"default_timeout"`
	Targets         []Target `yaml:"targets"`
}

// Load reads and validates a config file from disk.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Parse validates raw YAML bytes into a Config, applying defaults.
func Parse(raw []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if err := c.normalize(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) normalize() error {
	if c.DefaultInterval <= 0 {
		c.DefaultInterval = Duration(defaultInterval)
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = Duration(defaultTimeout)
	}
	if len(c.Targets) == 0 {
		return errors.New("config defines no targets")
	}

	seen := make(map[string]bool, len(c.Targets))
	for i := range c.Targets {
		t := &c.Targets[i]
		if t.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("duplicate target name %q", t.Name)
		}
		seen[t.Name] = true

		if t.Interval <= 0 {
			t.Interval = c.DefaultInterval
		}
		if t.Timeout <= 0 {
			t.Timeout = c.DefaultTimeout
		}

		switch t.Type {
		case "http":
			if t.URL == "" {
				return fmt.Errorf("target %q: url is required for an http check", t.Name)
			}
			if t.Method == "" {
				t.Method = "GET"
			}
			if t.ExpectStatus == 0 {
				t.ExpectStatus = 200
			}
		case "dns":
			if t.Host == "" {
				return fmt.Errorf("target %q: host is required for a dns check", t.Name)
			}
		case "tls":
			if t.Address == "" {
				return fmt.Errorf("target %q: address (host:port) is required for a tls check", t.Name)
			}
		default:
			return fmt.Errorf("target %q: unknown type %q (want http, dns, or tls)", t.Name, t.Type)
		}
	}
	return nil
}
