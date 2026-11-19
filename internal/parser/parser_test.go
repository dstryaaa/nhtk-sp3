package parser

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseLine_Valid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Record
	}{
		{
			name: "обычная строка со всеми полями",
			raw:  "2026-09-01T00:08:59Z | WARN | auth-api | AUTH_LOGIN_OK | User session created",
			want: Record{
				Timestamp: time.Date(2026, 9, 1, 0, 8, 59, 0, time.UTC),
				Level:     "WARN",
				Source:    "auth-api",
				EventCode: "AUTH_LOGIN_OK",
				Message:   "User session created",
			},
		},
		{
			name: "| внутри message не ломает разбор (SplitN с n=5)",
			raw:  "2026-09-06T11:10:00Z | ERROR | payment-worker | PAYMENT_TIMEOUT | message has | extra delimiter",
			want: Record{
				Timestamp: time.Date(2026, 9, 6, 11, 10, 0, 0, time.UTC),
				Level:     "ERROR",
				Source:    "payment-worker",
				EventCode: "PAYMENT_TIMEOUT",
				Message:   "message has | extra delimiter",
			},
		},
		{
			name: "пустой message допустим",
			raw:  "2026-09-08T14:00:00Z | ERROR | gateway | HTTP_502 | ",
			want: Record{
				Timestamp: time.Date(2026, 9, 8, 14, 0, 0, 0, time.UTC),
				Level:     "ERROR",
				Source:    "gateway",
				EventCode: "HTTP_502",
				Message:   "",
			},
		},
		{
			name: "level TRACE принимается",
			raw:  "2026-09-01T00:00:00Z | TRACE | svc | CODE | m",
			want: Record{
				Timestamp: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
				Level:     "TRACE",
				Source:    "svc",
				EventCode: "CODE",
				Message:   "m",
			},
		},
		{
			name: "level ERROR принимается",
			raw:  "2026-09-01T00:00:00Z | ERROR | svc | CODE | m",
			want: Record{
				Timestamp: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
				Level:     "ERROR",
				Source:    "svc",
				EventCode: "CODE",
				Message:   "m",
			},
		},
		{
			name: "пробелы вокруг полей обрезаются",
			raw:  "  2026-09-01T00:00:00Z   |   INFO   |   svc   |   CODE   |   msg   ",
			want: Record{
				Timestamp: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
				Level:     "INFO",
				Source:    "svc",
				EventCode: "CODE",
				Message:   "msg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseLine(tt.raw, 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Timestamp.Equal(tt.want.Timestamp) {
				t.Errorf("Timestamp = %v, want %v", got.Timestamp, tt.want.Timestamp)
			}
			if got.Level != tt.want.Level {
				t.Errorf("Level = %q, want %q", got.Level, tt.want.Level)
			}
			if got.Source != tt.want.Source {
				t.Errorf("Source = %q, want %q", got.Source, tt.want.Source)
			}
			if got.EventCode != tt.want.EventCode {
				t.Errorf("EventCode = %q, want %q", got.EventCode, tt.want.EventCode)
			}
			if got.Message != tt.want.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.want.Message)
			}
		})
	}
}

func TestParseLine_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "пустая строка",
			raw:     "",
			wantErr: "expected 5 fields",
		},
		{
			name:    "только 4 поля (нет message)",
			raw:     "2026-09-05T11:10:00Z | ERROR | payment-worker | PAYMENT_TIMEOUT",
			wantErr: "expected 5 fields",
		},
		{
			name:    "только 2 поля",
			raw:     "2026-09-28T20:28:18Z | INFO",
			wantErr: "expected 5 fields",
		},
		{
			name:    "битый timestamp",
			raw:     "not-a-timestamp | ERROR | orders-api | ORDER_REJECTED | timestamp is invalid",
			wantErr: "invalid timestamp",
		},
		{
			name:    "timestamp без T и Z",
			raw:     "2026-09-04 09:00:00 | WARN | inventory-api | STOCK_LOW | timestamp lacks ISO 8601",
			wantErr: "invalid timestamp",
		},
		{
			name:    "неизвестный level NOTICE",
			raw:     "2026-09-02T08:12:00Z | NOTICE | gateway | HTTP_502 | unsupported level",
			wantErr: "invalid level",
		},
		{
			name:    "пустой level",
			raw:     "2026-09-07T14:00:00Z |  | gateway | HTTP_502 | empty level",
			wantErr: "invalid level",
		},
		{
			name:    "строчный level error",
			raw:     "2026-09-01T00:00:00Z | error | svc | CODE | m",
			wantErr: "invalid level",
		},
		{
			name:    "пустой source",
			raw:     "2026-09-01T00:00:00Z | INFO |  | CODE | m",
			wantErr: "empty source",
		},
		{
			name:    "пустой eventCode",
			raw:     "2026-09-03T09:00:00Z | INFO | auth-api |  | missing event code",
			wantErr: "empty eventCode",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseLine(tt.raw, 42)
			if err == nil {
				t.Fatalf("expected error, got record: %+v", got)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			if pe.Line != 42 {
				t.Errorf("Line = %d, want 42", pe.Line)
			}
			if pe.Raw != tt.raw {
				t.Errorf("Raw = %q, want %q", pe.Raw, tt.raw)
			}
			if !strings.Contains(pe.Err.Error(), tt.wantErr) {
				t.Errorf("Err = %q, want substring %q", pe.Err.Error(), tt.wantErr)
			}
		})
	}
}
