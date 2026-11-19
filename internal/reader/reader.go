package reader

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/dstryaaa/nhtk-sp3/internal/parser"
)

// Summary — счётчики по результатам чтения файла.
type Summary struct {
	TotalLines     int                 // всего строк прочитано (включая пустые и битые)
	ValidLines     int                 // успешно распарсено
	InvalidLines   int                 // не удалось распарсить
	SkippedBlank   int                 // пустые строки (пропущены молча)
	InvalidSamples []parser.ParseError // первые N ошибок (для отчёта)
}

// maxInvalidSamples — сколько ошибок хранить в памяти.
const maxInvalidSamples = 20

// ProcessFile читает файл построчно и для каждой валидной записи
// вызывает fn. Не хранит все строки в памяти.
//
// Возвращает Summary со счётчиками и error только если файл
// не удалось открыть/прочитать
func ProcessFile(path string, fn func(rec *parser.Record)) (Summary, error) {
	var s Summary

	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	lineNum := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return s, err
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" && errors.Is(err, io.EOF) {
			break
		}

		lineNum++
		s.TotalLines++

		if line == "" {
			s.SkippedBlank++
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		rec, perr := parser.ParseLine(line, lineNum)
		if perr != nil {
			s.InvalidLines++
			if len(s.InvalidSamples) < maxInvalidSamples {
				var pe *parser.ParseError
				if errors.As(perr, &pe) {
					s.InvalidSamples = append(s.InvalidSamples, *pe)
				}
			}
		} else {
			s.ValidLines++
			fn(rec)
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return s, nil
}
