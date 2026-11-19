package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// WriteText пишет человекочитаемый отчёт.
func WriteText(w io.Writer, r Report) error {
	// Input
	if _, err := fmt.Fprintf(w, "Input: %s\n", r.Input.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Total lines: %d\n", r.Input.TotalLines); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Valid lines: %d\n", r.Input.ValidLines); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Invalid lines: %d\n", r.Input.InvalidLines); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Skipped blank: %d\n", r.Input.SkippedBlank); err != nil {
		return err
	}
	// Filter
	if _, err := fmt.Fprintf(w, "\nFilter: from=%s to=%s level=%s source=%s\n",
		orDash(r.Filter.From), orDash(r.Filter.To),
		orDash(r.Filter.Level), orDash(r.Filter.Source)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Matched: %d\n", r.Matched); err != nil {
		return err
	}

	// By level
	if _, err := fmt.Fprintln(w, "\nBy level:"); err != nil {
		return err
	}
	if len(r.ByLevel) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	} else {
		for _, k := range sortedKeys(r.ByLevel) {
			if _, err := fmt.Fprintf(w, "  %-8s %d\n", k, r.ByLevel[k]); err != nil {
				return err
			}
		}
	}

	// By source
	if _, err := fmt.Fprintln(w, "\nBy source:"); err != nil {
		return err
	}
	if len(r.BySource) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	} else {
		for _, k := range sortedKeys(r.BySource) {
			if _, err := fmt.Fprintf(w, "  %-16s %d\n", k, r.BySource[k]); err != nil {
				return err
			}
		}
	}

	// Top errors
	if _, err := fmt.Fprintln(w, "\nTop errors:"); err != nil {
		return err
	}
	if len(r.TopErrors) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	} else {
		for i, ce := range r.TopErrors {
			if _, err := fmt.Fprintf(w, "  %d. %-16s %d\n", i+1, ce.Code, ce.Count); err != nil {
				return err
			}
		}
	}

	// Invalid samples
	if _, err := fmt.Fprintln(w, "\nInvalid samples:"); err != nil {
		return err
	}
	if len(r.InvalidSamples) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	} else {
		for _, il := range r.InvalidSamples {
			if _, err := fmt.Fprintf(w, "  line %d: %s\n", il.Line, il.Reason); err != nil {
				return err
			}
		}
	}

	return nil
}

// WriteJSON пишет отчёт в JSON с отступами.
func WriteJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteCSV пишет отчёт в CSV «long format»:
// section,key,value
func WriteCSV(w io.Writer, r Report) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"section", "key", "value"})

	// input
	_ = cw.Write([]string{"input", "total_lines", strconv.Itoa(r.Input.TotalLines)})
	_ = cw.Write([]string{"input", "valid_lines", strconv.Itoa(r.Input.ValidLines)})
	_ = cw.Write([]string{"input", "invalid_lines", strconv.Itoa(r.Input.InvalidLines)})
	_ = cw.Write([]string{"input", "skipped_blank", strconv.Itoa(r.Input.SkippedBlank)})

	// matched
	_ = cw.Write([]string{"matched", "", strconv.Itoa(r.Matched)})

	// by_level
	for _, k := range sortedKeys(r.ByLevel) {
		_ = cw.Write([]string{"by_level", k, strconv.Itoa(r.ByLevel[k])})
	}

	// by_source
	for _, k := range sortedKeys(r.BySource) {
		_ = cw.Write([]string{"by_source", k, strconv.Itoa(r.BySource[k])})
	}

	// top_errors
	for _, ce := range r.TopErrors {
		_ = cw.Write([]string{"top_errors", ce.Code, strconv.Itoa(ce.Count)})
	}

	// invalid_samples
	for _, inv := range r.InvalidSamples {
		_ = cw.Write([]string{"invalid_samples", strconv.Itoa(inv.Line), inv.Reason})
	}

	return cw.Error()
}

// sortedKeys возвращает ключи map в лексикографическом порядке.
// Нужен для детерминированного вывода: порядок ключей в map
// в Go случаен, и без сортировки отчёт «прыгает» между запусками.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// orDash возвращает "-" для пустой строки, иначе саму строку.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
