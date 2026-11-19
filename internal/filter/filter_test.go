package filter

import (
	"testing"
	"time"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

// helper: создать запись с заданным временем, level, source.
func rec(ts time.Time, level, source string) *parser.Record {
	return &parser.Record{
		Timestamp: ts,
		Level:     level,
		Source:    source,
		EventCode: "CODE",
		Message:   "msg",
	}
}

// helper: указатель на время.
func tp(t time.Time) *time.Time { return &t }

func TestFilter_Match(t *testing.T) {
	// базовое время для тестов
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	before := base.Add(-time.Nanosecond)
	after := base.Add(time.Nanosecond)

	tests := []struct {
		name   string
		filter Filter
		rec    *parser.Record
		want   bool
	}{
		// --- пустой фильтр ---
		{
			name:   "пустой фильтр пропускает всё",
			filter: Filter{},
			rec:    rec(base, "INFO", "auth-api"),
			want:   true,
		},

		// --- from: включительно ---
		{
			name:   "from: запись ровно в from проходит",
			filter: Filter{From: tp(base)},
			rec:    rec(base, "INFO", "auth-api"),
			want:   true,
		},
		{
			name:   "from: запись на 1ns раньше from не проходит",
			filter: Filter{From: tp(base)},
			rec:    rec(before, "INFO", "auth-api"),
			want:   false,
		},
		{
			name:   "from: запись на 1ns позже from проходит",
			filter: Filter{From: tp(base)},
			rec:    rec(after, "INFO", "auth-api"),
			want:   true,
		},

		// --- to: исключительно ---
		{
			name:   "to: запись ровно в to не проходит",
			filter: Filter{To: tp(base)},
			rec:    rec(base, "INFO", "auth-api"),
			want:   false,
		},
		{
			name:   "to: запись на 1ns раньше to проходит",
			filter: Filter{To: tp(base)},
			rec:    rec(before, "INFO", "auth-api"),
			want:   true,
		},
		{
			name:   "to: запись на 1ns позже to не проходит",
			filter: Filter{To: tp(base)},
			rec:    rec(after, "INFO", "auth-api"),
			want:   false,
		},

		// --- from + to вместе ---
		{
			name:   "from+to: запись внутри интервала проходит",
			filter: Filter{From: tp(base.Add(-time.Hour)), To: tp(base.Add(time.Hour))},
			rec:    rec(base, "INFO", "auth-api"),
			want:   true,
		},
		{
			name:   "from+to: запись ровно в from проходит",
			filter: Filter{From: tp(base), To: tp(base.Add(time.Hour))},
			rec:    rec(base, "INFO", "auth-api"),
			want:   true,
		},
		{
			name:   "from+to: запись ровно в to не проходит",
			filter: Filter{From: tp(base.Add(-time.Hour)), To: tp(base)},
			rec:    rec(base, "INFO", "auth-api"),
			want:   false,
		},

		// --- level ---
		{
			name:   "level: совпадает",
			filter: Filter{Level: "ERROR"},
			rec:    rec(base, "ERROR", "auth-api"),
			want:   true,
		},
		{
			name:   "level: не совпадает",
			filter: Filter{Level: "ERROR"},
			rec:    rec(base, "INFO", "auth-api"),
			want:   false,
		},
		{
			name:   "level: регистрозависимо (error != ERROR)",
			filter: Filter{Level: "ERROR"},
			rec:    rec(base, "error", "auth-api"),
			want:   false,
		},

		// --- source ---
		{
			name:   "source: совпадает",
			filter: Filter{Source: "auth-api"},
			rec:    rec(base, "INFO", "auth-api"),
			want:   true,
		},
		{
			name:   "source: не совпадает",
			filter: Filter{Source: "auth-api"},
			rec:    rec(base, "INFO", "gateway"),
			want:   false,
		},
		{
			name:   "source: регистрозависимо",
			filter: Filter{Source: "auth-api"},
			rec:    rec(base, "INFO", "AUTH-API"),
			want:   false,
		},

		// --- сочетание ---
		{
			name:   "все условия совпадают",
			filter: Filter{From: tp(base.Add(-time.Hour)), To: tp(base.Add(time.Hour)), Level: "ERROR", Source: "gateway"},
			rec:    rec(base, "ERROR", "gateway"),
			want:   true,
		},
		{
			name:   "все кроме source",
			filter: Filter{From: tp(base.Add(-time.Hour)), To: tp(base.Add(time.Hour)), Level: "ERROR", Source: "gateway"},
			rec:    rec(base, "ERROR", "auth-api"),
			want:   false,
		},
		{
			name:   "все кроме level",
			filter: Filter{From: tp(base.Add(-time.Hour)), To: tp(base.Add(time.Hour)), Level: "ERROR", Source: "gateway"},
			rec:    rec(base, "INFO", "gateway"),
			want:   false,
		},
		{
			name:   "все кроме времени (раньше from)",
			filter: Filter{From: tp(base), To: tp(base.Add(time.Hour)), Level: "ERROR", Source: "gateway"},
			rec:    rec(before, "ERROR", "gateway"),
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.filter.Match(tt.rec)
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
