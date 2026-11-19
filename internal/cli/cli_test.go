package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLog создаёт временный лог-файл и возвращает путь.
func writeLog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return path
}

const sampleLog = `2026-09-01T00:08:59Z | INFO | auth-api | AUTH_LOGIN_OK | ok
2026-09-01T00:20:18Z | ERROR | payment-worker | PAYMENT_TIMEOUT | timeout
2026-09-01T00:30:00Z | ERROR | gateway | HTTP_502 | bad gateway
2026-09-01T00:40:00Z | WARN | auth-api | AUTH_TOKEN_EXPIRED | expired
not-a-timestamp | ERROR | orders-api | ORDER_REJECTED | bad
`

func TestRun_TextToStdout(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	err := Run([]string{"-input", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run error: %v (stderr: %s)", err, stderr.String())
	}
	out := stdout.String()
	for _, c := range []string{
		"Input: " + path,
		"Total lines: 5",
		"Valid lines: 4",
		"Invalid lines: 1",
		"Matched: 4",
		"By level:",
		"Top errors:",
		"line 5:",
	} {
		if !strings.Contains(out, c) {
			t.Errorf("output missing %q\n%s", c, out)
		}
	}
}

func TestRun_JSON(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	if err := Run([]string{"-input", path, "-format", "json"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if v["matched"].(float64) != 4 {
		t.Errorf("matched = %v, want 4", v["matched"])
	}
	// snake_case ключи в filter
	if !strings.Contains(stdout.String(), `"from"`) {
		t.Errorf("JSON missing snake_case key \"from\":\n%s", stdout.String())
	}
}

func TestRun_CSV(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	if err := Run([]string{"-input", path, "-format", "csv"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	rows, err := csv.NewReader(&stdout).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("too few CSV rows: %d", len(rows))
	}
	if rows[0][0] != "section" {
		t.Errorf("header = %v", rows[0])
	}
}

func TestRun_FilterLevel(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	if err := Run([]string{"-input", path, "-level", "ERROR"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Matched: 2") {
		t.Errorf("expected Matched: 2 for level ERROR, got:\n%s", stdout.String())
	}
}

func TestRun_FilterSource(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	if err := Run([]string{"-input", path, "-source", "gateway"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Matched: 1") {
		t.Errorf("expected Matched: 1 for source gateway, got:\n%s", stdout.String())
	}
}

func TestRun_FilterTimeRange(t *testing.T) {
	path := writeLog(t, sampleLog)
	var stdout, stderr bytes.Buffer

	// from включительно (00:20:18), to исключительно (00:40:00)
	// попадут: 00:20:18 (ERROR), 00:30:00 (ERROR)
	err := Run([]string{
		"-input", path,
		"-from", "2026-09-01T00:20:18Z",
		"-to", "2026-09-01T00:40:00Z",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Matched: 2") {
		t.Errorf("expected Matched: 2 for time range, got:\n%s", stdout.String())
	}
}

func TestRun_OutputFile(t *testing.T) {
	path := writeLog(t, sampleLog)
	outPath := filepath.Join(t.TempDir(), "report.json")
	var stdout, stderr bytes.Buffer

	if err := Run([]string{"-input", path, "-format", "json", "-output", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty when -output is set, got: %s", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("output file is not valid JSON: %v", err)
	}
}

func TestRun_Errors(t *testing.T) {
	path := writeLog(t, sampleLog)
	emptyPath := writeLog(t, "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing -input",
			args:    []string{},
			wantErr: "input is required",
		},
		{
			name:    "nonexistent file",
			args:    []string{"-input", filepath.Join(t.TempDir(), "nope.log")},
			wantErr: "read",
		},
		{
			name:    "empty file",
			args:    []string{"-input", emptyPath},
			wantErr: "is empty",
		},
		{
			name:    "invalid -format",
			args:    []string{"-input", path, "-format", "xml"},
			wantErr: "invalid format",
		},
		{
			name:    "negative -top",
			args:    []string{"-input", path, "-top", "-1"},
			wantErr: "top must be >= 0",
		},
		{
			name:    "invalid -from",
			args:    []string{"-input", path, "-from", "not-a-time"},
			wantErr: "invalid -from",
		},
		{
			name:    "invalid -to",
			args:    []string{"-input", path, "-to", "not-a-time"},
			wantErr: "invalid -to",
		},
		{
			name: "from >= to",
			args: []string{
				"-input", path,
				"-from", "2026-09-02T00:00:00Z",
				"-to", "2026-09-01T00:00:00Z",
			},
			wantErr: "from must be before to",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			err := Run(tt.args, &stdout, &stderr)
			if err == nil {
				t.Fatalf("expected error, got nil (stdout: %s)", stdout.String())
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
