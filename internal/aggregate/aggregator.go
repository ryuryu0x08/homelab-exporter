package aggregate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sort"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

const sourceLabelName = "homelab_source"

// GatherStatus describes whether required sources produced a usable scrape.
type GatherStatus string

const (
	GatherStatusSuccess     GatherStatus = "success"
	GatherStatusUnavailable GatherStatus = "unavailable"
)

type sourceResult struct {
	source   config.Source
	body     []byte
	duration time.Duration
	err      error
}

// Aggregator scrapes and merges configured Prometheus sources.
type Aggregator struct {
	scraper Scraper
	logger  *log.Logger
}

// New constructs an Aggregator. A nil logger discards weak-dependency errors.
func New(scraper Scraper, logger *log.Logger) *Aggregator {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Aggregator{scraper: scraper, logger: logger}
}

// Gather concurrently scrapes sources and returns valid Prometheus text.
func (a *Aggregator) Gather(ctx context.Context, sources []config.Source) ([]byte, GatherStatus) {
	results := a.scrapeAll(ctx, sources)
	families := make(map[string]*dto.MetricFamily)
	status := GatherStatusSuccess

	for index := range results {
		result := &results[index]
		if result.err != nil {
			a.logger.Printf("Aggregator.Gather source=%q scrape failed: %v", result.source.Name, result.err)
			if result.source.Dependency == config.DependencyRequired {
				status = GatherStatusUnavailable
			}
			continue
		}

		parsed, err := parseSource(result.source.Name, result.body)
		if err != nil {
			a.logger.Printf("Aggregator.Gather source=%q parse failed: %v", result.source.Name, err)
			result.err = err
			if result.source.Dependency == config.DependencyRequired {
				status = GatherStatusUnavailable
			}
			continue
		}
		err = mergeFamilies(families, parsed)
		if err != nil {
			a.logger.Printf("Aggregator.Gather source=%q merge failed: %v", result.source.Name, err)
			result.err = err
			if result.source.Dependency == config.DependencyRequired {
				status = GatherStatusUnavailable
			}
			continue
		}
	}

	addSourceStatusFamilies(families, results)
	body, err := encodeFamilies(families)
	if err != nil {
		a.logger.Printf("Aggregator.Gather encode failed: %v", err)
		return []byte{}, GatherStatusUnavailable
	}
	return body, status
}

func (a *Aggregator) scrapeAll(ctx context.Context, sources []config.Source) []sourceResult {
	results := make([]sourceResult, len(sources))
	var waitGroup sync.WaitGroup
	for index, source := range sources {
		waitGroup.Add(1)
		go func(index int, source config.Source) {
			defer waitGroup.Done()
			start := time.Now()
			body, err := a.scraper.Scrape(ctx, source)
			results[index] = sourceResult{source: source, body: body, duration: time.Since(start), err: err}
		}(index, source)
	}
	waitGroup.Wait()
	return results
}

func parseSource(sourceName string, body []byte) (map[string]*dto.MetricFamily, error) {
	parser := expfmt.TextParser{}
	families, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse exposition: %w", err)
	}
	for _, family := range families {
		for _, metric := range family.Metric {
			err = addSourceLabel(metric, sourceName)
			if err != nil {
				return nil, err
			}
		}
	}
	return families, nil
}

func addSourceLabel(metric *dto.Metric, sourceName string) error {
	for _, pair := range metric.Label {
		if pair.GetName() == sourceLabelName {
			return fmt.Errorf("metric already contains reserved label %q", sourceLabelName)
		}
	}
	name := sourceLabelName
	value := sourceName
	metric.Label = append(metric.Label, &dto.LabelPair{Name: &name, Value: &value})
	sort.Slice(metric.Label, func(left, right int) bool {
		return metric.Label[left].GetName() < metric.Label[right].GetName()
	})
	return nil
}

func mergeFamilies(destination, source map[string]*dto.MetricFamily) error {
	for name, incoming := range source {
		existing, exists := destination[name]
		if !exists {
			continue
		}
		if existing.GetType() != incoming.GetType() {
			return fmt.Errorf("metric family %q type conflict", name)
		}
		if existing.GetHelp() != incoming.GetHelp() {
			return fmt.Errorf("metric family %q help conflict", name)
		}
	}
	for name, incoming := range source {
		existing, exists := destination[name]
		if !exists {
			destination[name] = incoming
			continue
		}
		existing.Metric = append(existing.Metric, incoming.Metric...)
	}
	return nil
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
