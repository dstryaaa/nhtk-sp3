package report

import (
	"github.com/dstryaaa/nhtk-sp3/internal/agg"
	"github.com/dstryaaa/nhtk-sp3/internal/reader"
)

// Params — параметры фильтра, как их задал пользователь.
type Params struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Level  string `json:"level"`
	Source string `json:"source"`
}

// InvalidLine — одна повреждённая строка для отчёта.
type InvalidLine struct {
	Line   int    `json:"line"`
	Raw    string `json:"raw"`
	Reason string `json:"reason"`
}

// Report — полный отчёт: чтение + фильтрация + агрегация.
type Report struct {
	Input struct {
		Path         string `json:"path"`
		TotalLines   int    `json:"total_lines"`
		ValidLines   int    `json:"valid_lines"`
		InvalidLines int    `json:"invalid_lines"`
		SkippedBlank int    `json:"skipped_blank"`
	} `json:"input"`

	Filter  Params `json:"filter"`
	Matched int    `json:"matched"`

	ByLevel   map[string]int  `json:"by_level"`
	BySource  map[string]int  `json:"by_source"`
	TopErrors []agg.CodeCount `json:"top_errors"`

	InvalidSamples []InvalidLine `json:"invalid_samples"`
}

// Build собирает Report из результатов чтения и агрегации.
func Build(path string, rs reader.Summary, as agg.Summary, p Params) Report {
	r := Report{}
	r.Input.Path = path
	r.Input.TotalLines = rs.TotalLines
	r.Input.ValidLines = rs.ValidLines
	r.Input.InvalidLines = rs.InvalidLines
	r.Input.SkippedBlank = rs.SkippedBlank

	r.Filter = p
	r.Matched = as.Matched
	r.ByLevel = as.ByLevel
	r.BySource = as.BySource
	r.TopErrors = as.TopErrors

	r.InvalidSamples = make([]InvalidLine, 0, len(rs.InvalidSamples))
	for _, pe := range rs.InvalidSamples {
		r.InvalidSamples = append(r.InvalidSamples, InvalidLine{
			Line:   pe.Line,
			Raw:    pe.Raw,
			Reason: pe.Err.Error(),
		})
	}
	return r
}
