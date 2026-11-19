package filter

import (
	"time"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

// Filter — набор условий фильтрации.
type Filter struct {
	From   *time.Time
	To     *time.Time
	Level  string
	Source string
}

// Match возвращает true, если запись проходит все заданные условия (AND).
func (f Filter) Match(rec *parser.Record) bool {
	if f.From != nil && rec.Timestamp.Before(*f.From) {
		return false
	}
	if f.To != nil && !rec.Timestamp.Before(*f.To) {
		return false
	}
	if f.Level != "" && rec.Level != f.Level {
		return false
	}
	if f.Source != "" && rec.Source != f.Source {
		return false
	}
	return true
}
