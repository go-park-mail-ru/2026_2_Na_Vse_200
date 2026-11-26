# Мониторинг

Prometheus каждые 5 секунд собирает метрики API и Node Exporter. Grafana берёт эти данные из Prometheus и загружает дашборд из репозитория при запуске.

## Запуск

Из корня проекта в Bash с Linux Docker Engine, в том числе в WSL с Docker Engine внутри Ubuntu:

```bash
docker compose -f compose.yaml -f compose.monitoring.yaml up --build --wait
```

Откройте [дашборд](http://127.0.0.1:3001/d/api-monitoring). Для первого входа на локальном стенде используются `admin` / `admin`. Перед первым запуском пароль можно задать через `GRAFANA_ADMIN_PASSWORD`, порт — через `GRAFANA_PORT`.

Grafana доступна на `127.0.0.1`, а Prometheus и эндпоинты метрик — внутри сети Compose. CPU, память и корневой диск относятся к Linux-хосту Docker Engine. При запуске внутри WSL это показатели среды WSL.

## Проверка

Выполните серию запросов к главной и странице несуществующего исполнителя:

```bash
for i in $(seq 1 60); do
  curl -fsS -o /dev/null http://127.0.0.1:8080/api/v1/home || break
  curl -sS -o /dev/null http://127.0.0.1:8080/api/v1/artists/00000000-0000-0000-0000-000000000001
  sleep 0.25
done
```

На дашборде появятся запросы в секунду, задержка p95 и ответы `404`. Выберите маршрут `/api/v1/artists/{id}`, чтобы увидеть только запросы к исполнителю. Служебные `/health` и `/metrics` исключены из графиков API.
