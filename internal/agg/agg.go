package agg

import (
	"sort"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

// CodeCount — пара «код ошибки — количество».
type CodeCount struct {
	Code  string
	Count int
}

// Summary — результат агрегации.
type Summary struct {
	Matched     int
	ByLevel     map[string]int
	BySource    map[string]int
	ByErrorCode map[string]int
	TopErrors   []CodeCount
}

// Aggregator собирает статистику по записям.
type Aggregator struct {
	matched     int
	byLevel     map[string]int
	bySource    map[string]int
	byErrorCode map[string]int
}

// New создаёт пустой агрегатор.
func New() *Aggregator {

	return &Aggregator{
		byLevel:     make(map[string]int),
		bySource:    make(map[string]int),
		byErrorCode: make(map[string]int),
	}
}

// Add учитывает одну запись. Вызывается только для записей, прошедших фильтр.
func (a *Aggregator) Add(rec *parser.Record) {
	a.matched++
	a.byLevel[rec.Level]++
	a.bySource[rec.Source]++
	if rec.Level == "ERROR" {
		a.byErrorCode[rec.EventCode]++
	}
}

// Summary возвращает итоговую сводку с top-N кодами ошибок.
// topN <= 0 означает «все коды».
func (a *Aggregator) Summary(topN int) Summary {

	s := Summary{
		Matched:     a.matched,
		ByLevel:     a.byLevel,
		BySource:    a.bySource,
		ByErrorCode: a.byErrorCode,
	}
	top := make([]CodeCount, 0, len(a.byErrorCode))
	for code, count := range a.byErrorCode {
		top = append(top, CodeCount{
			Code:  code,
			Count: count,
		})
	}

	sort.Slice(top, func(i, j int) bool {
		if top[i].Count != top[j].Count {
			return top[i].Count > top[j].Count
		}
		return top[i].Code < top[j].Code
	})

	if topN > 0 && len(top) > topN {
		top = top[:topN]
	}
	s.TopErrors = top

	return s
}
