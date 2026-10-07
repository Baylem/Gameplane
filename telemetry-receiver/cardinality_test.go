package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestIngestBoundsDistinctVersionLabels(t *testing.T) {
	s := newServer(config{})
	ingest := func(version string) {
		t.Helper()
		body := fmt.Sprintf(`{"version":%q,"servers":0,"templates":0}`, version)
		w := httptest.NewRecorder()
		s.ingest(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ingest", strings.NewReader(body)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("ingest %q: status %d", version, w.Code)
		}
	}
	for i := range 160 {
		ingest(fmt.Sprintf("1.0.%d", i))
	}
	ingest("1.0.0") // A retained version stays identifiable after saturation.
	ingest("<invalid>")
	families, err := s.reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != "gameplane_telemetry_reports_total" {
			continue
		}
		counts := make(map[string]float64)
		for _, metric := range family.Metric {
			counts[metric.Label[0].GetValue()] = metric.GetCounter().GetValue()
		}
		if len(counts) != 130 || counts["1.0.0"] != 2 || counts["other"] != 32 || counts["invalid"] != 1 {
			t.Fatalf("unexpected bounded counters: labels=%d retained=%v other=%v invalid=%v", len(counts), counts["1.0.0"], counts["other"], counts["invalid"])
		}
		return
	}
	t.Fatal("reports metric missing")
}

func TestConcurrentIngestKeepsVersionBudget(t *testing.T) {
	s := newServer(config{})
	var workers sync.WaitGroup
	for i := range 256 {
		workers.Go(func() {
			body := fmt.Sprintf(`{"version":"2.0.%d","servers":0,"templates":0}`, i)
			w := httptest.NewRecorder()
			s.ingest(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ingest", strings.NewReader(body)))
			if w.Code != http.StatusNoContent {
				t.Errorf("ingest status %d", w.Code)
			}
		})
	}
	workers.Wait()
	families, err := s.reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != "gameplane_telemetry_reports_total" {
			continue
		}
		var total float64
		for _, metric := range family.Metric {
			total += metric.GetCounter().GetValue()
		}
		if len(family.Metric) != 129 || total != 256 {
			t.Fatalf("concurrent reports: labels=%d total=%v", len(family.Metric), total)
		}
		return
	}
	t.Fatal("reports metric missing")
}
