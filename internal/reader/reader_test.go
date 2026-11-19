package reader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

// writeFile создаёт временный файл с заданным содержимым и возвращает путь.
// t.TempDir() автоматически удаляет папку после теста.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

// valid1 / valid2 / invalid — переиспользуемые строки для тестов.
const (
	valid1  = "2026-09-01T00:08:59Z | WARN | auth-api | AUTH_LOGIN_OK | User session created"
	valid2  = "2026-09-01T00:20:18Z | ERROR | payment-worker | PAYMENT_TIMEOUT | Payment provider timed out"
	invalid = "not-a-timestamp | ERROR | orders-api | ORDER_REJECTED | bad"
)

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		want        Summary
		wantFnCalls int
		wantErr     bool
	}{
		{
			name:        "только валидные, \\n в конце",
			content:     valid1 + "\n" + valid2 + "\n",
			want:        Summary{TotalLines: 2, ValidLines: 2},
			wantFnCalls: 2,
		},
		{
			name:        "только валидные, без \\n в конце",
			content:     valid1 + "\n" + valid2,
			want:        Summary{TotalLines: 2, ValidLines: 2},
			wantFnCalls: 2,
		},
		{
			name:        "валидные + битые, до конца",
			content:     valid1 + "\n" + invalid + "\n" + valid2 + "\n",
			want:        Summary{TotalLines: 3, ValidLines: 2, InvalidLines: 1},
			wantFnCalls: 2,
		},
		{
			name:        "только битые",
			content:     invalid + "\n" + invalid + "\n",
			want:        Summary{TotalLines: 2, InvalidLines: 2},
			wantFnCalls: 0,
		},
		{
			name:        "пустая строка в середине",
			content:     valid1 + "\n" + "\n" + valid2 + "\n",
			want:        Summary{TotalLines: 3, ValidLines: 2, SkippedBlank: 1},
			wantFnCalls: 2,
		},
		{
			name:        "CRLF переводы строк",
			content:     valid1 + "\r\n",
			want:        Summary{TotalLines: 1, ValidLines: 1},
			wantFnCalls: 1,
		},
		{
			name:        "пустой файл",
			content:     "",
			want:        Summary{},
			wantFnCalls: 0,
		},
		{
			name:    "несуществующий файл",
			content: "", // не используется
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var path string
			if tt.wantErr {
				path = filepath.Join(t.TempDir(), "nope.log")
			} else {
				path = writeFile(t, tt.content)
			}

			fnCalls := 0
			got, err := ProcessFile(path, func(rec *parser.Record) {
				fnCalls++
				if rec == nil {
					t.Errorf("fn called with nil record")
				}
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.TotalLines != tt.want.TotalLines {
				t.Errorf("TotalLines = %d, want %d", got.TotalLines, tt.want.TotalLines)
			}
			if got.ValidLines != tt.want.ValidLines {
				t.Errorf("ValidLines = %d, want %d", got.ValidLines, tt.want.ValidLines)
			}
			if got.InvalidLines != tt.want.InvalidLines {
				t.Errorf("InvalidLines = %d, want %d", got.InvalidLines, tt.want.InvalidLines)
			}
			if got.SkippedBlank != tt.want.SkippedBlank {
				t.Errorf("SkippedBlank = %d, want %d", got.SkippedBlank, tt.want.SkippedBlank)
			}
			if fnCalls != tt.wantFnCalls {
				t.Errorf("fn calls = %d, want %d", fnCalls, tt.wantFnCalls)
			}
		})
	}
}

// TestProcessFile_InvalidSamplesCapped проверяет, что память не растёт
// бесконечно: храним только первые maxInvalidSamples ошибок.
func TestProcessFile_InvalidSamplesCapped(t *testing.T) {
	t.Parallel()

	var content string
	for i := 0; i < 25; i++ {
		content += "bad line\n"
	}
	path := writeFile(t, content)

	got, err := ProcessFile(path, func(rec *parser.Record) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InvalidLines != 25 {
		t.Errorf("InvalidLines = %d, want 25", got.InvalidLines)
	}
	if len(got.InvalidSamples) != maxInvalidSamples {
		t.Errorf("len(InvalidSamples) = %d, want %d", len(got.InvalidSamples), maxInvalidSamples)
	}
}

// TestProcessFile_InvalidLineNumber проверяет, что номер ошибочной строки
// соответствует реальной позиции в файле (требование ТЗ).
func TestProcessFile_InvalidLineNumber(t *testing.T) {
	t.Parallel()

	path := writeFile(t, valid1+"\n"+invalid+"\n"+valid2+"\n")

	got, err := ProcessFile(path, func(rec *parser.Record) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.InvalidSamples) != 1 {
		t.Fatalf("len(InvalidSamples) = %d, want 1", len(got.InvalidSamples))
	}
	if got.InvalidSamples[0].Line != 2 {
		t.Errorf("invalid line number = %d, want 2", got.InvalidSamples[0].Line)
	}
	if got.InvalidSamples[0].Raw != invalid {
		t.Errorf("Raw = %q, want %q", got.InvalidSamples[0].Raw, invalid)
	}
}

// TestProcessFile_OnlyInvalid проверяет, что файл из одних битых строк
// обрабатывается до конца без падения (требование ТЗ).
func TestProcessFile_OnlyInvalid(t *testing.T) {
	t.Parallel()

	content := invalid + "\n" + invalid + "\n" + invalid + "\n"
	path := writeFile(t, content)

	got, err := ProcessFile(path, func(rec *parser.Record) {
		t.Errorf("fn should not be called for invalid-only file")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalLines != 3 {
		t.Errorf("TotalLines = %d, want 3", got.TotalLines)
	}
	if got.InvalidLines != 3 {
		t.Errorf("InvalidLines = %d, want 3", got.InvalidLines)
	}
	if got.ValidLines != 0 {
		t.Errorf("ValidLines = %d, want 0", got.ValidLines)
	}
}
