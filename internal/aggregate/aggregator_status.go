package aggregate

import dto "github.com/prometheus/client_model/go"

const (
	sourceUpMetricName       = "homelab_exporter_source_up"
	sourceDurationMetricName = "homelab_exporter_source_scrape_duration_seconds"
	sourceStatusLabelName    = "source"
)

func addSourceStatusFamilies(families map[string]*dto.MetricFamily, results []sourceResult) {
	families[sourceUpMetricName] = newSourceStatusFamily(
		sourceUpMetricName,
		"Whether the source was scraped and merged successfully.",
		results,
		func(result sourceResult) float64 {
			if result.err != nil {
				return 0
			}
			return 1
		},
	)
	families[sourceDurationMetricName] = newSourceStatusFamily(
		sourceDurationMetricName,
		"Time spent scraping the source in seconds.",
		results,
		func(result sourceResult) float64 { return result.duration.Seconds() },
	)
}

func newSourceStatusFamily(name, help string, results []sourceResult, value func(sourceResult) float64) *dto.MetricFamily {
	metricType := dto.MetricType_GAUGE
	family := &dto.MetricFamily{Name: &name, Help: &help, Type: &metricType}
	for _, result := range results {
		labelName := sourceStatusLabelName
		labelValue := result.source.Name
		metricValue := value(result)
		family.Metric = append(family.Metric, &dto.Metric{
			Label: []*dto.LabelPair{{Name: &labelName, Value: &labelValue}},
			Gauge: &dto.Gauge{Value: &metricValue},
		})
	}
	return family
}
