package agg

import (
	"reflect"
	"testing"
	"time"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

func rec(level, source, code string) *parser.Record {
	return &parser.Record{
		Timestamp: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Level:     level,
		Source:    source,
		EventCode: code,
		Message:   "m",
	}
}

func TestAggregator_Empty(t *testing.T) {
	a := New()
	s := a.Summary(5)

	if s.Matched != 0 {
		t.Errorf("Matched = %d, want 0", s.Matched)
	}
	if len(s.ByLevel) != 0 {
		t.Errorf("ByLevel = %v, want empty", s.ByLevel)
	}
	if len(s.BySource) != 0 {
		t.Errorf("BySource = %v, want empty", s.BySource)
	}
	if len(s.ByErrorCode) != 0 {
		t.Errorf("ByErrorCode = %v, want empty", s.ByErrorCode)
	}
	if s.TopErrors == nil {
		t.Errorf("TopErrors = nil, want empty non-nil slice")
	}
	if len(s.TopErrors) != 0 {
		t.Errorf("len(TopErrors) = %d, want 0", len(s.TopErrors))
	}
}

func TestAggregator_Counts(t *testing.T) {
	a := New()
	a.Add(rec("INFO", "auth-api", "AUTH_LOGIN_OK"))
	a.Add(rec("INFO", "auth-api", "AUTH_LOGIN_OK"))
	a.Add(rec("ERROR", "gateway", "HTTP_502"))
	a.Add(rec("ERROR", "gateway", "HTTP_502"))
	a.Add(rec("ERROR", "gateway", "HTTP_502"))
	a.Add(rec("WARN", "inventory-api", "STOCK_LOW"))

	s := a.Summary(0)

	if s.Matched != 6 {
		t.Errorf("Matched = %d, want 6", s.Matched)
	}
	wantLevel := map[string]int{"INFO": 2, "ERROR": 3, "WARN": 1}
	if !reflect.DeepEqual(s.ByLevel, wantLevel) {
		t.Errorf("ByLevel = %v, want %v", s.ByLevel, wantLevel)
	}
	wantSource := map[string]int{"auth-api": 2, "gateway": 3, "inventory-api": 1}
	if !reflect.DeepEqual(s.BySource, wantSource) {
		t.Errorf("BySource = %v, want %v", s.BySource, wantSource)
	}
	// только ERROR попадают в ByErrorCode
	wantErrCode := map[string]int{"HTTP_502": 3}
	if !reflect.DeepEqual(s.ByErrorCode, wantErrCode) {
		t.Errorf("ByErrorCode = %v, want %v", s.ByErrorCode, wantErrCode)
	}
}

func TestAggregator_OnlyErrorGoesToByErrorCode(t *testing.T) {
	a := New()
	a.Add(rec("INFO", "s", "CODE_INFO"))
	a.Add(rec("WARN", "s", "CODE_WARN"))
	a.Add(rec("DEBUG", "s", "CODE_DEBUG"))
	a.Add(rec("TRACE", "s", "CODE_TRACE"))
	a.Add(rec("ERROR", "s", "CODE_ERR"))

	s := a.Summary(0)

	if len(s.ByErrorCode) != 1 {
		t.Fatalf("ByErrorCode = %v, want only CODE_ERR", s.ByErrorCode)
	}
	if s.ByErrorCode["CODE_ERR"] != 1 {
		t.Errorf("CODE_ERR = %d, want 1", s.ByErrorCode["CODE_ERR"])
	}
}

func TestAggregator_TopN(t *testing.T) {
	// Коды с разными счётчиками:
	// A=5, B=3, C=3, D=1
	// Tie между B и C разрешается по коду возрастанию: B < C.
	a := New()
	for i := 0; i < 5; i++ {
		a.Add(rec("ERROR", "s", "A"))
	}
	for i := 0; i < 3; i++ {
		a.Add(rec("ERROR", "s", "B"))
	}
	for i := 0; i < 3; i++ {
		a.Add(rec("ERROR", "s", "C"))
	}
	a.Add(rec("ERROR", "s", "D"))

	tests := []struct {
		name string
		topN int
		want []CodeCount
	}{
		{
			name: "topN=0 — все коды",
			topN: 0,
			want: []CodeCount{{"A", 5}, {"B", 3}, {"C", 3}, {"D", 1}},
		},
		{
			name: "topN=2 — только A и B",
			topN: 2,
			want: []CodeCount{{"A", 5}, {"B", 3}},
		},
		{
			name: "topN=3 — A, B, C (tie-break B<C)",
			topN: 3,
			want: []CodeCount{{"A", 5}, {"B", 3}, {"C", 3}},
		},
		{
			name: "topN=100 — все, обрезки нет",
			topN: 100,
			want: []CodeCount{{"A", 5}, {"B", 3}, {"C", 3}, {"D", 1}},
		},
		{
			name: "topN отрицательный — все",
			topN: -1,
			want: []CodeCount{{"A", 5}, {"B", 3}, {"C", 3}, {"D", 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := a.Summary(tt.topN)
			if !reflect.DeepEqual(s.TopErrors, tt.want) {
				t.Errorf("TopErrors = %v, want %v", s.TopErrors, tt.want)
			}
		})
	}
}

// TestAggregator_DeterministicTieBreak проверяет, что при равных счётчиках
// порядок стабильный: по коду возрастанию. Это требование ТЗ
// «детерминированный результат».
func TestAggregator_DeterministicTieBreak(t *testing.T) {
	a := New()
	// Все по 1, порядок должен быть алфавитным: AAA, BBB, CCC, DDD
	a.Add(rec("ERROR", "s", "DDD"))
	a.Add(rec("ERROR", "s", "BBB"))
	a.Add(rec("ERROR", "s", "AAA"))
	a.Add(rec("ERROR", "s", "CCC"))

	s := a.Summary(0)
	want := []CodeCount{{"AAA", 1}, {"BBB", 1}, {"CCC", 1}, {"DDD", 1}}
	if !reflect.DeepEqual(s.TopErrors, want) {
		t.Errorf("TopErrors = %v, want %v", s.TopErrors, want)
	}
}
