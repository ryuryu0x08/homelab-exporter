package aggregate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

type fakeScraper struct {
	bodies map[string]string
	errs   map[string]error
}

func (f fakeScraper) Scrape(_ context.Context, source config.Source) ([]byte, error) {
	if err := f.errs[source.Name]; err != nil {
		return nil, err
	}
	return []byte(f.bodies[source.Name]), nil
}

func TestGatherMergesFamiliesAndAddsSourceLabel(t *testing.T) {
	metric := "# HELP shared_total test metric\n# TYPE shared_total counter\nshared_total 1\n"
	aggregator := New(fakeScraper{bodies: map[string]string{"one": metric, "two": metric}}, nil)
	sources := []config.Source{
		{Name: "one", URL: "http://one/metrics", Dependency: config.DependencyOptional},
		{Name: "two", URL: "http://two/metrics", Dependency: config.DependencyOptional},
	}

	body, status := aggregator.Gather(context.Background(), sources)
	if status != GatherStatusSuccess {
		t.Fatalf("status=%q, want success", status)
	}
	text := string(body)
	for _, want := range []string{`shared_total{homelab_source="one"} 1`, `shared_total{homelab_source="two"} 1`} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestGatherOptionalFailureKeepsHealthyMetrics(t *testing.T) {
	aggregator := New(fakeScraper{
		bodies: map[string]string{"healthy": "healthy_metric 7\n"},
		errs:   map[string]error{"dynamic": errors.New("connection refused")},
	}, nil)
	sources := []config.Source{
		{Name: "healthy", URL: "http://healthy/metrics", Dependency: config.DependencyRequired},
		{Name: "dynamic", URL: "http://dynamic/metrics", Dependency: config.DependencyOptional},
	}

	body, status := aggregator.Gather(context.Background(), sources)
	if status != GatherStatusSuccess {
		t.Fatalf("status=%q, want success", status)
	}
	text := string(body)
	if !strings.Contains(text, `homelab_exporter_source_up{source="healthy"} 1`) {
		t.Fatalf("healthy source status missing:\n%s", text)
	}
	if !strings.Contains(text, `homelab_exporter_source_up{source="dynamic"} 0`) {
		t.Fatalf("failed source status missing:\n%s", text)
	}
	if !strings.Contains(text, `healthy_metric{homelab_source="healthy"} 7`) {
		t.Fatalf("healthy metric missing:\n%s", text)
	}
}

func TestGatherRequiredFailureReturnsUnavailable(t *testing.T) {
	aggregator := New(fakeScraper{errs: map[string]error{"required": errors.New("offline")}}, nil)
	sources := []config.Source{{Name: "required", URL: "http://required/metrics", Dependency: config.DependencyRequired}}

	_, status := aggregator.Gather(context.Background(), sources)
	if status != GatherStatusUnavailable {
		t.Fatalf("status=%q, want unavailable", status)
	}
}

func TestGatherRejectsConflictingMetricFamilies(t *testing.T) {
	aggregator := New(fakeScraper{bodies: map[string]string{
		"one": "# TYPE conflict gauge\nconflict 1\n",
		"two": "# TYPE conflict counter\nconflict 2\n",
	}}, nil)
	sources := []config.Source{
		{Name: "one", Dependency: config.DependencyRequired},
		{Name: "two", Dependency: config.DependencyRequired},
	}

	_, status := aggregator.Gather(context.Background(), sources)
	if status != GatherStatusUnavailable {
		t.Fatalf("status=%q, want unavailable", status)
	}
}
