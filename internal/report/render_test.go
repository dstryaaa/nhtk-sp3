package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/dstryaaa/nhtk-sp3/internal/agg"
)

// sampleReport возвращает наполненный Report для тестов.
func sampleReport() Report {
	r := Report{}
	r.Input.Path = "logs.log"
	r.Input.TotalLines = 100
	r.Input.ValidLines = 95
	r.Input.InvalidLines = 5
	r.Input.SkippedBlank = 2
	r.Filter = Params{From: "2026-09-01T00:00:00Z", Level: "ERROR"}
	r.Matched = 42
	r.ByLevel = map[string]int{"ERROR": 42, "WARN": 1, "INFO": 3}
	r.BySource = map[string]int{"gateway": 20, "payment-worker": 22}
	r.TopErrors = []agg.CodeCount{
		{Code: "HTTP_502", Count: 20},
		{Code: "PAYMENT_TIMEOUT", Count: 22},
	}
	r.InvalidSamples = []InvalidLine{
		{Line: 36, Raw: "bad", Reason: "invalid timestamp: ..."},
		{Line: 100, Raw: "bad2", Reason: "invalid level: \"NOTICE\""},
	}
	return r
}

func TestWriteText(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleReport()); err != nil {
		t.Fatalf("WriteText error: %v", err)
	}
	out := buf.String()

	// 1. ключевые подстроки
	checks := []string{
		"Input: logs.log",
		"Total lines: 100",
		"Valid lines: 95",
		"Invalid lines: 5",
		"Skipped blank: 2",
		"from=2026-09-01T00:00:00Z",
		"to=-",
		"level=ERROR",
		"source=-",
		"Matched: 42",
		"By level:",
		"By source:",
		"Top errors:",
		"Invalid samples:",
		"line 36: invalid timestamp",
		"line 100: invalid level",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("output missing %q\n---\n%s", c, out)
		}
	}

	// 2. by_level отсортирован: ищем только внутри секции "By level:"
	byLevel := out[strings.Index(out, "By level:"):]
	iError := strings.Index(byLevel, "ERROR")
	iInfo := strings.Index(byLevel, "INFO")
	iWarn := strings.Index(byLevel, "WARN")
	if !(iError >= 0 && iInfo >= 0 && iWarn >= 0 && iError < iInfo && iInfo < iWarn) {
		t.Errorf("by_level not sorted: ERROR=%d INFO=%d WARN=%d\n%s",
			iError, iInfo, iWarn, byLevel)
	}

	// 3. by_source отсортирован: gateway < payment-worker
	bySource := out[strings.Index(out, "By source:"):]
	iGateway := strings.Index(bySource, "gateway")
	iPayment := strings.Index(bySource, "payment-worker")
	if !(iGateway >= 0 && iPayment >= 0 && iGateway < iPayment) {
		t.Errorf("by_source not sorted: gateway=%d payment=%d\n%s",
			iGateway, iPayment, bySource)
	}
}

func TestWriteText_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, Report{}); err != nil {
		t.Fatalf("WriteText error: %v", err)
	}
	out := buf.String()
	for _, c := range []string{"(none)", "Matched: 0", "Total lines: 0"} {
		if !strings.Contains(out, c) {
			t.Errorf("output missing %q\n%s", c, out)
		}
	}
}

func TestWriteJSON_RoundTrip(t *testing.T) {
	r := sampleReport()
	var buf bytes.Buffer
	if err := WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var got Report
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v\n%s", err, buf.String())
	}

	if got.Input.Path != r.Input.Path {
		t.Errorf("Path = %q, want %q", got.Input.Path, r.Input.Path)
	}
	if got.Input.TotalLines != r.Input.TotalLines {
		t.Errorf("TotalLines = %d, want %d", got.Input.TotalLines, r.Input.TotalLines)
	}
	if got.Matched != r.Matched {
		t.Errorf("Matched = %d, want %d", got.Matched, r.Matched)
	}
	if !reflect.DeepEqual(got.ByLevel, r.ByLevel) {
		t.Errorf("ByLevel = %v, want %v", got.ByLevel, r.ByLevel)
	}
	if !reflect.DeepEqual(got.BySource, r.BySource) {
		t.Errorf("BySource = %v, want %v", got.BySource, r.BySource)
	}
	if !reflect.DeepEqual(got.TopErrors, r.TopErrors) {
		t.Errorf("TopErrors = %v, want %v", got.TopErrors, r.TopErrors)
	}
	if !reflect.DeepEqual(got.InvalidSamples, r.InvalidSamples) {
		t.Errorf("InvalidSamples = %v, want %v", got.InvalidSamples, r.InvalidSamples)
	}
}

func TestWriteJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, Report{}); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, sampleReport()); err != nil {
		t.Fatalf("WriteCSV error: %v", err)
	}

	cr := csv.NewReader(&buf)
	rows, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("csv parse error: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("too few rows: %d", len(rows))
	}
	if !reflect.DeepEqual(rows[0], []string{"section", "key", "value"}) {
		t.Errorf("header = %v", rows[0])
	}

	type sk struct{ section, key string }
	idx := make(map[sk]string)
	sections := make(map[string]int)
	for _, row := range rows[1:] {
		if len(row) != 3 {
			t.Fatalf("row has %d columns, want 3: %v", len(row), row)
		}
		idx[sk{row[0], row[1]}] = row[2]
		sections[row[0]]++
	}

	// input
	if idx[sk{"input", "total_lines"}] != "100" {
		t.Errorf("total_lines = %q, want 100", idx[sk{"input", "total_lines"}])
	}
	if idx[sk{"input", "invalid_lines"}] != "5" {
		t.Errorf("invalid_lines = %q, want 5", idx[sk{"input", "invalid_lines"}])
	}

	// matched
	if idx[sk{"matched", ""}] != "42" {
		t.Errorf("matched = %q, want 42", idx[sk{"matched", ""}])
	}

	// by_level: 3 строки, ERROR=42
	if sections["by_level"] != 3 {
		t.Errorf("by_level rows = %d, want 3", sections["by_level"])
	}
	if idx[sk{"by_level", "ERROR"}] != "42" {
		t.Errorf("by_level ERROR = %q, want 42", idx[sk{"by_level", "ERROR"}])
	}

	// by_source: 2 строки, ключи — источники
	if sections["by_source"] != 2 {
		t.Errorf("by_source rows = %d, want 2", sections["by_source"])
	}
	if idx[sk{"by_source", "gateway"}] != "20" {
		t.Errorf("by_source gateway = %q, want 20", idx[sk{"by_source", "gateway"}])
	}
	if _, ok := idx[sk{"by_source", "ERROR"}]; ok {
		t.Errorf("by_source contains level key ERROR — copy-paste bug not fixed")
	}

	// top_errors: 2 строки
	if sections["top_errors"] != 2 {
		t.Errorf("top_errors rows = %d, want 2", sections["top_errors"])
	}
	if idx[sk{"top_errors", "HTTP_502"}] != "20" {
		t.Errorf("top_errors HTTP_502 = %q, want 20", idx[sk{"top_errors", "HTTP_502"}])
	}

	// invalid_samples: 2 строки, ключ — номер строки
	if sections["invalid_samples"] != 2 {
		t.Errorf("invalid_samples rows = %d, want 2", sections["invalid_samples"])
	}
	if _, ok := idx[sk{"invalid_samples", "36"}]; !ok {
		t.Errorf("invalid_samples missing line 36")
	}
}

func TestWriteCSV_DeterministicOrder(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	if err := WriteCSV(&buf1, sampleReport()); err != nil {
		t.Fatal(err)
	}
	if err := WriteCSV(&buf2, sampleReport()); err != nil {
		t.Fatal(err)
	}
	if buf1.String() != buf2.String() {
		t.Errorf("CSV output not deterministic:\n---1---\n%s\n---2---\n%s",
			buf1.String(), buf2.String())
	}
}

// failingWriter всегда возвращает ошибку — для покрытия веток обработки ошибок.
type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errTest("write failed")
}

func TestWriteText_WriteError(t *testing.T) {
	if err := WriteText(failingWriter{}, sampleReport()); err == nil {
		t.Error("expected error from failing writer, got nil")
	}
}

func TestWriteJSON_WriteError(t *testing.T) {
	if err := WriteJSON(failingWriter{}, sampleReport()); err == nil {
		t.Error("expected error from failing writer, got nil")
	}
}

func TestWriteText_PartiallyEmpty(t *testing.T) {
	r := Report{}
	r.BySource = map[string]int{"gateway": 5}
	r.TopErrors = []agg.CodeCount{{Code: "HTTP_502", Count: 5}}

	var buf bytes.Buffer
	if err := WriteText(&buf, r); err != nil {
		t.Fatalf("WriteText error: %v", err)
	}
	out := buf.String()

	// by_level пуст -> (none), by_source непуст -> gateway
	byLevel := out[strings.Index(out, "By level:"):strings.Index(out, "By source:")]
	if !strings.Contains(byLevel, "(none)") {
		t.Errorf("by_level should be (none), got:\n%s", byLevel)
	}
	if strings.Contains(byLevel, "gateway") {
		t.Errorf("by_level should not contain gateway")
	}

	// invalid_samples пуст -> (none)
	if !strings.Contains(out, "Invalid samples:") || !strings.Contains(out, "(none)") {
		t.Errorf("invalid_samples should show (none)")
	}
}

func TestWriteCSV_EmptySections(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, Report{}); err != nil {
		t.Fatalf("WriteCSV error: %v", err)
	}
	cr := csv.NewReader(&buf)
	rows, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("csv parse error: %v", err)
	}
	// header + 4 input + 1 matched = 6 строк (by_level/by_source/top_errors/invalid_samples пусты)
	if len(rows) != 6 {
		t.Errorf("rows = %d, want 6", len(rows))
	}
}
