#!/usr/bin/env bash
#
# Загружает переменные окружения из .env и запускает команду.
#
#   ./scripts/run.sh                       запустить сервер
#   ./scripts/run.sh go test ./...         выполнить команду с теми же настройками
#   ENV_FILE=.env.stand ./scripts/run.sh   взять другой файл настроек

set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="${ENV_FILE:-.env}"

if [ -f "$ENV_FILE" ]; then
	# set -a включает автоэкспорт: иначе переменные не дойдут до дочернего процесса.
	set -a
	# shellcheck source=/dev/null
	. "./$ENV_FILE"
	set +a
	echo "настройки загружены из $ENV_FILE"
else
	echo "файл $ENV_FILE не найден — используются значения по умолчанию" >&2
	echo "чтобы задать свои: cp .env.example .env" >&2
fi

if [ "$#" -eq 0 ]; then
	exec go run ./cmd/server
fi

# exec, чтобы сигнал остановки уходил напрямую серверу.
exec "$@"
