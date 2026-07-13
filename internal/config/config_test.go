package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseValidWindowsConfig(t *testing.T) {
	raw := `
platform = "windows"
listen = ":9836"
scrape_timeout = "5s"
max_body_bytes = 10485760

[[source]]
name = "windows"
url = "http://127.0.0.1:9182/metrics"
dependency = "required"
`

	config, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if config.Platform != PlatformWindows {
		t.Fatalf("platform=%q, want windows", config.Platform)
	}
	if config.ScrapeTimeout != 5*time.Second {
		t.Fatalf("timeout=%s, want 5s", config.ScrapeTimeout)
	}
}

func TestParseRejectsDuplicateSourceName(t *testing.T) {
	raw := `
platform = "windows"
listen = ":9836"
scrape_timeout = "5s"
max_body_bytes = 1024
[[source]]
name = "same"
url = "http://127.0.0.1:1/metrics"
dependency = "optional"
[[source]]
name = "same"
url = "http://127.0.0.1:2/metrics"
dependency = "optional"
`

	_, err := Parse([]byte(raw))
	if err == nil || !strings.Contains(err.Error(), "duplicate source") {
		t.Fatalf("err=%v, want duplicate source error", err)
	}
}

func TestParseRejectsUnknownDependency(t *testing.T) {
	raw := `
platform = "windows"
listen = ":9836"
scrape_timeout = "5s"
max_body_bytes = 1024
[[source]]
name = "bad"
url = "http://127.0.0.1:1/metrics"
dependency = "sometimes"
`

	_, err := Parse([]byte(raw))
	if err == nil || !strings.Contains(err.Error(), "dependency") {
		t.Fatalf("err=%v, want dependency error", err)
	}
}
