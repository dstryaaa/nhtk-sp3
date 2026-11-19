package parser

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Record - одна корректная запись лога.
type Record struct {
	Timestamp time.Time
	Level     string
	Source    string
	EventCode string
	Message   string
}

// ParseError - ошибка разбора конкретной строки
type ParseError struct {
	Line int
	Raw  string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %v", e.Line, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// допустимые уровни
var validLevels = map[string]bool{
	"TRACE": true,
	"DEBUG": true,
	"INFO":  true,
	"WARN":  true,
	"ERROR": true,
}

// ParseLine разбирает одну строку лога.
// lineNum — номер строки (1-based).
// Возвращает (*Record, nil) при успехе, (nil, *ParseError) при ошибке.
func ParseLine(raw string, lineNum int) (*Record, error) {

	// получаем поля
	parts := strings.SplitN(raw, "|", 5)
	if len(parts) != 5 {
		return nil, &ParseError{
			Line: lineNum,
			Raw:  raw,
			Err:  fmt.Errorf("expected 5 fields, got %d", len(parts)),
		}
	}

	// обрезаем пробелы у всех полей
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// timestamp
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return nil, &ParseError{
			Line: lineNum,
			Raw:  raw,
			Err:  fmt.Errorf("invalid timestamp: %w", err),
		}
	}

	// level
	if !validLevels[parts[1]] {
		return nil, &ParseError{
			Line: lineNum,
			Raw:  raw,
			Err:  fmt.Errorf("invalid level: %q", parts[1]),
		}
	}

	// source и eventCode обязательны
	if parts[2] == "" {
		return nil, &ParseError{
			Line: lineNum,
			Raw:  raw,
			Err:  errors.New("empty source"),
		}
	}
	if parts[3] == "" {
		return nil, &ParseError{
			Line: lineNum,
			Raw:  raw,
			Err:  errors.New("empty eventCode"),
		}
	}

	return &Record{
		Timestamp: ts,
		Level:     parts[1],
		Source:    parts[2],
		EventCode: parts[3],
		Message:   parts[4],
	}, nil
}
