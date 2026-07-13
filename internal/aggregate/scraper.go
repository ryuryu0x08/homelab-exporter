package aggregate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

// Scraper retrieves Prometheus exposition data from one source.
type Scraper interface {
	Scrape(ctx context.Context, source config.Source) ([]byte, error)
}

type httpScraper struct {
	client       *http.Client
	maxBodyBytes int64
}

// NewHTTPScraper constructs a bounded HTTP scraper.
func NewHTTPScraper(client *http.Client, maxBodyBytes int64) Scraper {
	return &httpScraper{client: client, maxBodyBytes: maxBodyBytes}
}

func (s *httpScraper) Scrape(ctx context.Context, source config.Source) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "text/plain; version=0.0.4")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request metrics: %w", err)
	}
	reader := io.LimitReader(response.Body, s.maxBodyBytes+1)
	body, err := io.ReadAll(reader)
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil {
		return nil, errors.Join(wrapReadError(err), wrapCloseError(closeErr))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	if int64(len(body)) > s.maxBodyBytes {
		return nil, fmt.Errorf("metrics response exceeds %d bytes", s.maxBodyBytes)
	}
	return body, nil
}

func wrapReadError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("read metrics response: %w", err)
}

func wrapCloseError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("close metrics response: %w", err)
}
