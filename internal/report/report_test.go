package report

import (
	"testing"

	"github.com/dstryaaa/nhtk-sp3/internal/agg"
	"github.com/dstryaaa/nhtk-sp3/internal/parser"
	"github.com/dstryaaa/nhtk-sp3/internal/reader"
)

func TestBuild(t *testing.T) {
	rs := reader.Summary{
		TotalLines:   100,
		ValidLines:   95,
		InvalidLines: 5,
		SkippedBlank: 2,
		InvalidSamples: []parser.ParseError{
			{Line: 36, Raw: "bad", Err: errTest("invalid timestamp")},
			{Line: 100, Raw: "bad2", Err: errTest("invalid level")},
		},
	}
	as := agg.Summary{
		Matched:     42,
		ByLevel:     map[string]int{"ERROR": 42},
		BySource:    map[string]int{"gateway": 20, "payment-worker": 22},
		ByErrorCode: map[string]int{"HTTP_502": 20},
		TopErrors:   []agg.CodeCount{{Code: "HTTP_502", Count: 20}},
	}
	p := Params{From: "2026-09-01T00:00:00Z", Level: "ERROR"}

	got := Build("logs.log", rs, as, p)

	// Input
	if got.Input.Path != "logs.log" {
		t.Errorf("Path = %q, want logs.log", got.Input.Path)
	}
	if got.Input.TotalLines != 100 {
		t.Errorf("TotalLines = %d, want 100", got.Input.TotalLines)
	}
	if got.Input.ValidLines != 95 {
		t.Errorf("ValidLines = %d, want 95", got.Input.ValidLines)
	}
	if got.Input.InvalidLines != 5 {
		t.Errorf("InvalidLines = %d, want 5", got.Input.InvalidLines)
	}
	if got.Input.SkippedBlank != 2 {
		t.Errorf("SkippedBlank = %d, want 2", got.Input.SkippedBlank)
	}

	// Filter
	if got.Filter != p {
		t.Errorf("Filter = %+v, want %+v", got.Filter, p)
	}

	// Matched + agg
	if got.Matched != 42 {
		t.Errorf("Matched = %d, want 42", got.Matched)
	}
	if len(got.ByLevel) != 1 || got.ByLevel["ERROR"] != 42 {
		t.Errorf("ByLevel = %v, want {ERROR:42}", got.ByLevel)
	}
	if len(got.BySource) != 2 || got.BySource["gateway"] != 20 {
		t.Errorf("BySource = %v", got.BySource)
	}
	if len(got.TopErrors) != 1 || got.TopErrors[0].Code != "HTTP_502" {
		t.Errorf("TopErrors = %v", got.TopErrors)
	}

	// InvalidSamples: проверяем маппинг Line/Raw/Reason
	if len(got.InvalidSamples) != 2 {
		t.Fatalf("len(InvalidSamples) = %d, want 2", len(got.InvalidSamples))
	}
	if got.InvalidSamples[0].Line != 36 {
		t.Errorf("InvalidSamples[0].Line = %d, want 36", got.InvalidSamples[0].Line)
	}
	if got.InvalidSamples[0].Raw != "bad" {
		t.Errorf("InvalidSamples[0].Raw = %q, want bad", got.InvalidSamples[0].Raw)
	}
	if got.InvalidSamples[0].Reason != "invalid timestamp" {
		t.Errorf("InvalidSamples[0].Reason = %q, want invalid timestamp",
			got.InvalidSamples[0].Reason)
	}
}

func TestBuild_EmptyInvalidSamples(t *testing.T) {
	rs := reader.Summary{TotalLines: 10, ValidLines: 10}
	as := agg.Summary{
		Matched:  10,
		ByLevel:  map[string]int{"INFO": 10},
		BySource: map[string]int{"svc": 10},
	}
	got := Build("x.log", rs, as, Params{})

	if got.InvalidSamples == nil {
		t.Errorf("InvalidSamples = nil, want empty non-nil slice")
	}
	if len(got.InvalidSamples) != 0 {
		t.Errorf("len(InvalidSamples) = %d, want 0", len(got.InvalidSamples))
	}
}

// errTest — минимальная реализация error для тестов.
type errTest string

func (e errTest) Error() string { return string(e) }
