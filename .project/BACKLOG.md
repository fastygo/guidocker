# Backlog and deployment notes

Да, именно это вы и хотите добавить — и это хорошая идея для прод-процесса: не вставлять вручную `compose.yaml`, а класть только URL репозитория и давать системе самой подтянуть его, собрать/сконфигурить и запустить.

Сейчас в текущей версии это **не сделано по этой схеме**, потому что `Create App` умеет только прямой YAML:

```dashboard/interfaces/paas_handlers.go:56-63
case http.MethodPost:
    payload, ok := h.decodeAppPayload(w, r)
    ...
    app, err := h.appUseCase.CreateApp(r.Context(), payload.Name, payload.ComposeYAML)
```

И сам `payload` сейчас только с полями `name` и `compose_yaml`:

```dashboard/interfaces/paas_handlers.go:14-17
type appPayload struct {
    Name        string `json:"name"`
    ComposeYAML string `json:"compose_yaml"`
}
```

А сервис создания приложения принимает ровно те же два поля:

```dashboard/usecase/app/service.go:34-40
func (s *Service) CreateApp(ctx context.Context, name, composeYAML string) (*domain.App, error) {
    if err := validateAppInput(name, composeYAML); err != nil { ... }
```

Поэтому да: без доработки UI/API/сервиса ссылку на репозиторий в `create app` пока не засунуть как «запускаемый источник».

---

Как сделать в `create app` как вы хотите (очень практичный MVP):

1. Оставить текущий режим как есть (compose_yaml), добавить второй режим `import_repo`.
2. В UI (`dashboard/views/compose.html`) добавить вкладку/переключатель:
   - `Repo URL`
   - `Branch` (опц.)
   - `Compose file path` (например, `compose.yml`, `docker-compose.yml`)
   - флажок `Deploy now`
3. Добавить новый endpoint, например:
   - `POST /api/apps/import`
   - payload:
   ```json
   {
     "name": "my-site",
     "repo_url": "https://github.com/user/site.git",
     "branch": "main",
     "compose_path": "docker-compose.yml",
     "auto_deploy": true
   }
   ```
4. В сервисе добавить новый метод (или расширить `CreateApp` через новый путь):
   - Клонировать репозиторий в `STACKS_DIR/<app-id>`
   - Найти compose:
     - если есть `compose_path` и файл найден — читать его
     - если нет — считать сайт статическим и сгенерировать compose на лету (например nginx + томом на клонированный каталог)
   - Сохранить app в BoltDB с `ComposeYAML`
   - Если `auto_deploy=true` — сразу дернуть `DeployApp(id)`
5. Защитить безопасность:
   - только `https://` и валидный URL
   - лимитить время `git clone`
   - для приватных репо — передавать `repo_auth_token` через безопасный backend-параметр, а не в UI
   - очищать временные директории при ошибках

Пример автогенерации compose для «просто index.html + style.css»:
```yaml
version: "3.9"
services:
  web:
    image: nginx:alpine
    ports:
      - "3000:80"
    volumes:
      - ./:/usr/share/nginx/html:ro
```

---

Про порт 80:
- Если запускать на порту 80 в `compose.yaml` из этой админки, конфликтов не избежать при нескольких приложениях (все хотят 80).
- Для одного приложения на сервере — можно ставить `80:80`.
- Для нескольких — лучше через обратный прокси (Nginx/Caddy) и давать каждому приложению внутренний порт.

---

Что лучше сделать следующим:
1. Если вам нужно быстро — я дам “минимум файлов + дифф” как паттерн импорта репо.
2. Если хотите, я тоже могу сразу расписать это как `Scanner/Import` задачу в вашем стиле: список файлов, интерфейсы, обработчики и сценарий деплоя.

Если хотите, я сейчас в следующем сообщении дам готовую схему именно для вашего кода `app.go / service.go / pas/handlers.go / compose.html` с конкретными шагами в стиле вашего проекта.