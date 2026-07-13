package windows

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

type fakeRunner struct {
	output []byte
	err    error
}

func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return f.output, f.err
}

func TestNVIDIAScraperExportsGPUValues(t *testing.T) {
	output := "0, GPU-one, NVIDIA GeForce RTX 5060 Ti, 7, 11, 16311, 8063, 8248, 36, 17.5, 180, 240, 405, 45\n"
	scraper := newNVIDIASMIScraper(fakeRunner{output: []byte(output)})

	body, err := scraper.Scrape(context.Background(), config.Source{Kind: config.SourceKindNVIDIASMI})
	if err != nil {
		t.Fatalf("Scrape() error=%v", err)
	}
	text := string(body)
	for _, expected := range []string{
		`homelab_nvidia_gpu_utilization_ratio{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 0.07`,
		`homelab_nvidia_gpu_memory_used_bytes{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 8.454668288e+09`,
		`homelab_nvidia_gpu_temperature_celsius{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 36`,
		`nvidia_smi_utilization_gpu_ratio{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 0.07`,
		`nvidia_smi_gpu_info{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 1`,
		`nvidia_smi_fan_speed_ratio{gpu="0",name="NVIDIA GeForce RTX 5060 Ti",uuid="GPU-one"} 0.45`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output missing %q:\n%s", expected, text)
		}
	}
}

func TestNVIDIAScraperPropagatesCommandFailure(t *testing.T) {
	scraper := newNVIDIASMIScraper(fakeRunner{err: errors.New("command failed")})
	_, err := scraper.Scrape(context.Background(), config.Source{Kind: config.SourceKindNVIDIASMI})
	if err == nil {
		t.Fatal("Scrape() error=nil, want command error")
	}
}

func TestNVIDIAScraperRejectsMalformedRow(t *testing.T) {
	scraper := newNVIDIASMIScraper(fakeRunner{output: []byte("0, too-short\n")})
	_, err := scraper.Scrape(context.Background(), config.Source{Kind: config.SourceKindNVIDIASMI})
	if err == nil {
		t.Fatal("Scrape() error=nil, want row error")
	}
}
