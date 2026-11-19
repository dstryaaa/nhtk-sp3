package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dstryaaa/nhtk-sp3/internal/agg"
	"github.com/dstryaaa/nhtk-sp3/internal/filter"
	"github.com/dstryaaa/nhtk-sp3/internal/parser"
	"github.com/dstryaaa/nhtk-sp3/internal/reader"
	"github.com/dstryaaa/nhtk-sp3/internal/report"
)

// Run — точка входа CLI. Возвращает error вместо os.Exit,
// чтобы её можно было тестировать.
func Run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("loganalyzer", flag.ContinueOnError)
	fs.SetOutput(stderr)

	input := fs.String("input", "", "path to log file (required)")
	fromStr := fs.String("from", "", "start time, RFC3339, inclusive")
	toStr := fs.String("to", "", "end time, RFC3339, exclusive")
	level := fs.String("level", "", "level filter (TRACE/DEBUG/INFO/WARN/ERROR)")
	source := fs.String("source", "", "source filter")
	topN := fs.Int("top", 5, "top-N error codes (0 = all)")
	format := fs.String("format", "text", "output format: text|json|csv")
	output := fs.String("output", "", "output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *input == "" {
		return fmt.Errorf("input is required")
	}

	switch *format {
	case "text", "json", "csv":
		// Формат валидный, продолжаем работу
	default:
		return fmt.Errorf("invalid format %q: must be one of text, json, csv", *format)
	}

	if *topN < 0 {
		return fmt.Errorf("top must be >= 0")
	}

	var fromT, toT *time.Time
	if *fromStr != "" {
		t, err := time.Parse(time.RFC3339, *fromStr)
		if err != nil {
			return fmt.Errorf("invalid -from %q: %w", *fromStr, err)
		}
		fromT = &t
	}

	if *toStr != "" {
		t, err := time.Parse(time.RFC3339, *toStr)
		if err != nil {
			return fmt.Errorf("invalid -to %q: %w", *toStr, err)
		}
		toT = &t
	}

	if fromT != nil && toT != nil && !fromT.Before(*toT) {
		return fmt.Errorf("from must be before to: %s >= %s",
			fromT.Format(time.RFC3339), toT.Format(time.RFC3339))
	}

	f := filter.Filter{
		From:   fromT,
		To:     toT,
		Level:  *level,
		Source: *source,
	}

	a := agg.New()
	readSum, err := reader.ProcessFile(*input, func(rec *parser.Record) {
		if f.Match(rec) {
			a.Add(rec)
		}
	})
	if err != nil {
		return fmt.Errorf("read %s: %w", *input, err)
	}

	if readSum.TotalLines == 0 {
		return fmt.Errorf("file %s is empty", *input)
	}

	params := report.Params{
		From:   *fromStr,
		To:     *toStr,
		Level:  *level,
		Source: *source,
	}
	rep := report.Build(*input, readSum, a.Summary(*topN), params)

	var w io.Writer = stdout
	if *output != "" {
		outFile, err := os.Create(*output)
		if err != nil {
			return fmt.Errorf("create %s: %w", *output, err)
		}
		defer outFile.Close()
		w = outFile
	}

	switch *format {
	case "text":
		return report.WriteText(w, rep)
	case "json":
		return report.WriteJSON(w, rep)
	case "csv":
		return report.WriteCSV(w, rep)
	}

	return nil
}
