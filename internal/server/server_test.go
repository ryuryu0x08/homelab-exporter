package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryuryu0x08/homelab-exporter/internal/aggregate"
)

type fakeGatherer struct {
	body   []byte
	status aggregate.GatherStatus
}

func (f fakeGatherer) Gather(context.Context) ([]byte, aggregate.GatherStatus) {
	return f.body, f.status
}

func TestHandlerServesMetrics(t *testing.T) {
	handler := New(fakeGatherer{body: []byte("sample 1\n"), status: aggregate.GatherStatusSuccess}, nil)
	request := httptest.NewRequest(http.MethodGet, metricsPath, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "sample 1\n" {
		t.Fatalf("body=%q", recorder.Body.String())
	}
}

func TestHandlerReturnsUnavailableForRequiredFailure(t *testing.T) {
	handler := New(fakeGatherer{status: aggregate.GatherStatusUnavailable}, nil)
	request := httptest.NewRequest(http.MethodGet, metricsPath, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := New(fakeGatherer{}, nil)
	request := httptest.NewRequest(http.MethodPost, metricsPath, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
