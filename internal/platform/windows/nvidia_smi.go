package windows

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

const (
	nvidiaSMIExecutable = "nvidia-smi.exe"
	queryArgument       = "--query-gpu=index,uuid,name,utilization.gpu,utilization.memory,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,clocks.current.graphics,clocks.current.memory"
	formatArgument      = "--format=csv,noheader,nounits"
	nvidiaFieldCount    = 13
	percentScale        = 100
	mebibyteBytes       = 1024 * 1024
	megahertzScale      = 1000 * 1000
)

type commandRunner interface {
	Run(ctx context.Context, executable string, arguments ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, executable string, arguments ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, executable, arguments...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %s: %w: %s", executable, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

// NVIDIASMIScraper collects metrics directly from the NVIDIA driver CLI.
type NVIDIASMIScraper struct {
	runner commandRunner
}

// NewNVIDIASMIScraper creates a native Windows NVIDIA collector.
func NewNVIDIASMIScraper() *NVIDIASMIScraper {
	return &NVIDIASMIScraper{runner: execRunner{}}
}

func newNVIDIASMIScraper(runner commandRunner) *NVIDIASMIScraper {
	return &NVIDIASMIScraper{runner: runner}
}

func (s *NVIDIASMIScraper) Scrape(ctx context.Context, _ config.Source) ([]byte, error) {
	output, err := s.runner.Run(ctx, nvidiaSMIExecutable, queryArgument, formatArgument)
	if err != nil {
		return nil, err
	}
	rows, err := csv.NewReader(bytes.NewReader(output)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse nvidia-smi CSV: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("nvidia-smi returned no GPUs")
	}
	return encodeGPURows(rows)
}

func encodeGPURows(rows [][]string) ([]byte, error) {
	families := newGPUFamilies()
	for index, row := range rows {
		if len(row) != nvidiaFieldCount {
			return nil, fmt.Errorf("nvidia-smi row %d has %d fields, want %d", index, len(row), nvidiaFieldCount)
		}
		err := appendGPU(families, row)
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi row %d: %w", index, err)
		}
	}
	return encodeFamilies(families)
}

func appendGPU(families map[string]*dto.MetricFamily, row []string) error {
	labels := gpuLabels(row[0], row[1], row[2])
	for _, descriptor := range gpuMetricDescriptors {
		value, err := strconv.ParseFloat(strings.TrimSpace(row[descriptor.field+3]), 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", descriptor.name, err)
		}
		value *= descriptor.scale
		families[descriptor.name].Metric = append(families[descriptor.name].Metric, gaugeMetric(labels, value))
	}
	families["nvidia_smi_gpu_info"].Metric = append(families["nvidia_smi_gpu_info"].Metric, gaugeMetric(labels, 1))
	return nil
}

func gpuLabels(index, uuid, name string) []*dto.LabelPair {
	return []*dto.LabelPair{
		labelPair("gpu", strings.TrimSpace(index)),
		labelPair("name", strings.TrimSpace(name)),
		labelPair("uuid", strings.TrimSpace(uuid)),
	}
}

func labelPair(name, value string) *dto.LabelPair {
	return &dto.LabelPair{Name: &name, Value: &value}
}

func gaugeMetric(labels []*dto.LabelPair, value float64) *dto.Metric {
	return &dto.Metric{Label: labels, Gauge: &dto.Gauge{Value: &value}}
}
