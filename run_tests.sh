#!/usr/bin/env bash
# run_tests.sh — одна команда: тесты + покрытие + HTML-отчёт.
# Использование: ./run_tests.sh
set -euo pipefail

cd "$(dirname "$0")"

echo "==> go test ./..."
go test ./... -coverprofile=coverage.out -covermode=atomic

echo
echo "==> покрытие по пакетам:"
go tool cover -func=coverage.out | tail -n 20

echo
echo "==> HTML-отчёт: coverage.html"
go tool cover -html=coverage.out -o coverage.html

echo
echo "Готово. Открой coverage.html в браузере:"
echo "  file://$(pwd)/coverage.html"