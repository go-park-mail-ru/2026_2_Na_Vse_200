# Мониторинг

Из корня проекта запустите:

```bash
docker compose -f compose.yaml -f compose.monitoring.yaml up --build --wait
```

Откройте [дашборд Grafana](http://127.0.0.1:3001/d/api-monitoring). Для первого локального входа используйте `admin` / `admin`. Начальный пароль можно задать через `GRAFANA_ADMIN_PASSWORD`.

Grafana доступна на `127.0.0.1:3001`. Prometheus каждые 5 секунд опрашивает API и Node Exporter внутри сети Compose. Графики ресурсов показывают Linux-среду, в которой работает Docker.

## Проверка

Отправьте из Bash серию запросов к главной и несуществующему исполнителю:

```bash
for i in $(seq 1 60); do
  curl -fsS -o /dev/null http://127.0.0.1:8080/api/v1/home || break
  curl -sS -o /dev/null http://127.0.0.1:8080/api/v1/artists/00000000-0000-0000-0000-000000000001
  sleep 0.25
done
```

На дашборде появятся трафик, p95 и ответы `404`. Фильтр «Маршрут» позволяет посмотреть одну ручку. `/health` и `/metrics` исключены из графиков API.
