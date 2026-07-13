package windows

import (
	"bytes"
	"fmt"
	"sort"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type gpuMetricDescriptor struct {
	name  string
	help  string
	scale float64
}

var gpuMetricDescriptors = []gpuMetricDescriptor{
	{name: "homelab_nvidia_gpu_utilization_ratio", help: "Current GPU compute utilization ratio.", scale: 1 / float64(percentScale)},
	{name: "homelab_nvidia_gpu_memory_utilization_ratio", help: "Current GPU memory controller utilization ratio.", scale: 1 / float64(percentScale)},
	{name: "homelab_nvidia_gpu_memory_total_bytes", help: "Total GPU memory in bytes.", scale: mebibyteBytes},
	{name: "homelab_nvidia_gpu_memory_used_bytes", help: "Used GPU memory in bytes.", scale: mebibyteBytes},
	{name: "homelab_nvidia_gpu_memory_free_bytes", help: "Free GPU memory in bytes.", scale: mebibyteBytes},
	{name: "homelab_nvidia_gpu_temperature_celsius", help: "Current GPU temperature in degrees Celsius.", scale: 1},
	{name: "homelab_nvidia_gpu_power_draw_watts", help: "Current GPU power draw in watts.", scale: 1},
	{name: "homelab_nvidia_gpu_power_limit_watts", help: "Configured GPU power limit in watts.", scale: 1},
	{name: "homelab_nvidia_gpu_graphics_clock_hertz", help: "Current GPU graphics clock in hertz.", scale: megahertzScale},
	{name: "homelab_nvidia_gpu_memory_clock_hertz", help: "Current GPU memory clock in hertz.", scale: megahertzScale},
}

func newGPUFamilies() map[string]*dto.MetricFamily {
	families := make(map[string]*dto.MetricFamily, len(gpuMetricDescriptors))
	metricType := dto.MetricType_GAUGE
	for _, descriptor := range gpuMetricDescriptors {
		name := descriptor.name
		help := descriptor.help
		families[name] = &dto.MetricFamily{Name: &name, Help: &help, Type: &metricType}
	}
	return families
}

func encodeFamilies(families map[string]*dto.MetricFamily) ([]byte, error) {
	names := make([]string, 0, len(families))
	for name := range families {
		names = append(names, name)
	}
	sort.Strings(names)

	var buffer bytes.Buffer
	for _, name := range names {
		_, err := expfmt.MetricFamilyToText(&buffer, families[name])
		if err != nil {
			return nil, fmt.Errorf("encode metric family %q: %w", name, err)
		}
	}
	return buffer.Bytes(), nil
}
