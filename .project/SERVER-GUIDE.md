# Server Upgrade Guide (Step-by-step)

Это пошаговый чеклист для обновления/переезда админки на проде без потери данных.

## 0) Подготовка

1. Подключитесь к серверу с доступом root (или sudo).
2. Убедитесь, что есть свободные порты и место на диске.
3. Скопируйте себе команды ниже в отдельный блокнот, чтобы выполнять последовательно.

```bash
export DASHBOARD_HOST="5.129.197.52"   # ваш IP
export OLD_NAME="dashboard"
export NEW_NAME="dashboard-new"
export IMAGE_NAME="paas-dashboard:latest"
export STACKS_DIR="/opt/stacks"
export DATA_DIR="/opt/stacks"
export BOLT_PATH="/opt/stacks/.paas.db"
```

## 1) Быстрая диагностика текущего состояния

1. Посмотреть текущий контейнер:
```bash
docker ps --filter name=$OLD_NAME --format 'table {{.Names}}\t{{.Image}}\t{{.Ports}}\t{{.Status}}'
```
2. Проверить, где лежит база и проекты:
```bash
ls -la "$STACKS_DIR"
```
3. Проверить, кто сейчас сидит на порту 3000:
```bash
ss -lntp | grep ':3000' || true
```

## 2) Резервное копирование (обязательно)

1. Остановите обновление, если нет свободного места.
2. Скопируйте базу и конфиг:
```bash
mkdir -p /opt/backups
ts=$(date +%F_%H%M%S)
tar -czf "/opt/backups/dashboard-state-$ts.tar.gz" "$DATA_DIR/.paas.db" "$STACKS_DIR"
ls -lh /opt/backups
```
3. Если у вас есть важные каталоги проектов внутри `$STACKS_DIR`, проверьте их содержимое:
```bash
find "$STACKS_DIR" -maxdepth 2 -type d -name '*'
```

## 3) Обновление исходного кода

Вариант А (рекомендуется): обновить в существующей директории.
```bash
cd /opt/dashboard-new  # или /opt/dashboard
git fetch --all
git checkout <branch-or-tag>   # например main
git pull --ff-only
```

Вариант Б (чистый update без изменения старого дерева): заново клонировать.
```bash
cd /opt
git clone <repo-url> dashboard-new
cd /opt/dashboard-new
git checkout <branch-or-tag>
```

## 4) Сборка image

Если есть make (быстрый путь):
```bash
cd /opt/dashboard-new
make docker-build IMAGE_NAME=paas-dashboard IMAGE_TAG=latest
```

Без make (fallback для минималистичных серверов):
```bash
cd /opt/dashboard-new
docker build -t paas-dashboard:latest .
```

## 5) Подготовить новый запуск в "тестовом" режиме

На этом этапе запускаем вторую копию на порту 3001, чтобы проверить всё до переключения.

```bash
docker rm -f "$NEW_NAME" 2>/dev/null || true
docker run -d \
  --name "$NEW_NAME" \
  --restart unless-stopped \
  --group-add "$(getent group docker | cut -d: -f3)" \
  -p 3001:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$STACKS_DIR:/opt/stacks" \
  -e SERVER_HOST=0.0.0.0 \
  -e PAAS_PORT=3000 \
  -e PAAS_ADMIN_USER=admin \
  -e PAAS_ADMIN_PASS='admin@123' \
  -e DASHBOARD_AUTH_DISABLED=false \
  -e STACKS_DIR=/opt/stacks \
  -e BOLT_DB_FILE=/opt/stacks/.paas.db \
  $IMAGE_NAME
```

## 6) Проверка "песочницы" новой версии

1. Проверить контейнер:
```bash
docker ps --filter name=$NEW_NAME --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
docker logs -n 80 "$NEW_NAME"
```
2. Проверить основные URL:
```bash
curl -I "http://$DASHBOARD_HOST:3001/"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3001/api/apps"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3001/api/scan"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3001/api/settings"
```
3. Проверить, что старые приложения видны и управляемы:
```bash
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3001/api/apps"
```
Ожидаемый результат: список приложений из старой БД + корректный формат JSON.
4. Проверить роутинг (если есть app с public_domain):
```bash
curl -I "http://$DASHBOARD_HOST:3001/scan"
```

## 7) Проверка API импорта и ключевых сценариев (рекомендуется)

1. Проверка создания/обновления конфигурации приложения (через API, если используете автоматизацию/CI):
```bash
curl -u admin:admin@123 -X PUT \
  -H "Content-Type: application/json" \
  -d '{"public_domain":"test.example.com","proxy_target_port":8080,"use_tls":false}' \
  http://$DASHBOARD_HOST:3001/api/apps/<app-id>/config
```
2. Проверка ручного renewal endpoint (если нужен минимальный проверочный прогон):
```bash
curl -u admin:admin@123 -X POST http://$DASHBOARD_HOST:3001/api/certificates/renew
```
3. Проверка очистки/безопасного удаления (безопасный сценарий):
```bash
curl -u admin:admin@123 -X DELETE http://$DASHBOARD_HOST:3001/api/apps/<app-id>
```
Если есть внешние зависимости, API вернёт предупреждение с ошибкой `manual cleanup required`.

## 8) Переключение на продакшн (safe cutover)

Вариант А (краткий простой): прямая замена контейнера.
```bash
docker rm -f "$OLD_NAME"
docker run -d \
  --name "$OLD_NAME" \
  --restart unless-stopped \
  --group-add "$(getent group docker | cut -d: -f3)" \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$STACKS_DIR:/opt/stacks" \
  -e SERVER_HOST=0.0.0.0 \
  -e PAAS_PORT=3000 \
  -e PAAS_ADMIN_USER=admin \
  -e PAAS_ADMIN_PASS='admin@123' \
  -e DASHBOARD_AUTH_DISABLED=false \
  -e STACKS_DIR=/opt/stacks \
  -e BOLT_DB_FILE=/opt/stacks/.paas.db \
  $IMAGE_NAME
```

Вариант B (переименовать старый + поднять новый под старым именем): удобнее для трассировки логов и rollback.
```bash
docker rename "$OLD_NAME" "${OLD_NAME}-old"
docker rm -f "${OLD_NAME}-old" 2>/dev/null || true
```
и затем запуск новой как `$OLD_NAME` по порту 3000, как в блоке выше.

## 9) Финальная верификация после cutover

1. Проверить доступ с обычного порта:
```bash
curl -I "http://$DASHBOARD_HOST:3000/"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3000/api/apps"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3000/api/scan"
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3000/api/settings"
```
2. Проверить существующие managed-приложения:
```bash
curl -u admin:admin@123 "http://$DASHBOARD_HOST:3000/api/apps"
```
3. Проверить nginx конфиг/маршруты (если включен прокси):
```bash
docker exec "$OLD_NAME" ls -l /etc/nginx/sites-enabled || true
```

## 10) Очистка старых артефактов и проверка журнала

1. Убедитесь, что новый контейнер стабилен хотя бы 5–10 минут:
```bash
docker logs --tail 200 "$OLD_NAME"
```
2. Если не нужна временная версия, удалить её:
```bash
docker rm -f "$NEW_NAME" 2>/dev/null || true
```
3. Проверить размер базы и стека:
```bash
du -sh "$BOLT_PATH" "$STACKS_DIR"
```

## 11) Rollback (если есть сбой)

Если после переключения есть регрессии:
```bash
docker rm -f "$OLD_NAME"
docker run -d \
  --name "$OLD_NAME" \
  --restart unless-stopped \
  --group-add "$(getent group docker | cut -d: -f3)" \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$STACKS_DIR:/opt/stacks" \
  -e SERVER_HOST=0.0.0.0 \
  -e PAAS_PORT=3000 \
  -e PAAS_ADMIN_USER=admin \
  -e PAAS_ADMIN_PASS='admin@123' \
  -e DASHBOARD_AUTH_DISABLED=false \
  -e STACKS_DIR=/opt/stacks \
  -e BOLT_DB_FILE=/opt/stacks/.paas.db \
  # image старой версии, который был до апдейта
  paas-dashboard:previous
```
Затем разберите логи и сравните с backup.

## 12) Что проверять после каждого релиза

- API: `GET /api/apps`, `GET /api/scan`, `GET /api/settings`
- UI открывается по `/`
- Старые приложения по-прежнему видны
- Удаление app без мусора (в частности проверка warning `manual cleanup`)
- Конфигурация TLS:
  - если `UseTLS` включён, должны быть корректные поля certbot в settings
  - при необходимости обновления сертификатов вручную: `POST /api/certificates/renew`

## 13) Чистка после деплоя

- Удалить неиспользуемые образы:
```bash
docker image prune -f
```
- Удалить старые бэкапы (по политике хранения):
```bash
find /opt/backups -name 'dashboard-state-*.tar.gz' -mtime +14 -delete
```