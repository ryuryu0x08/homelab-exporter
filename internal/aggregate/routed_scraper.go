package aggregate

import (
	"context"
	"fmt"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

type routedScraper struct {
	scrapers map[config.SourceKind]Scraper
}

// NewRoutedScraper dispatches each configured source to its collector type.
func NewRoutedScraper(httpScraper, nvidiaSMIScraper Scraper) Scraper {
	return &routedScraper{scrapers: map[config.SourceKind]Scraper{
		config.SourceKindHTTP:      httpScraper,
		config.SourceKindNVIDIASMI: nvidiaSMIScraper,
	}}
}

func (s *routedScraper) Scrape(ctx context.Context, source config.Source) ([]byte, error) {
	scraper, exists := s.scrapers[source.Kind]
	if !exists {
		return nil, fmt.Errorf("source kind %q has no scraper", source.Kind)
	}
	body, err := scraper.Scrape(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("scrape %s source: %w", source.Kind, err)
	}
	return body, nil
}
