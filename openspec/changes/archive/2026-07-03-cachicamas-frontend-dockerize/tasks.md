# Tasks: cachicamas-frontend-dockerize

> **Cambio**: `cachicamas-frontend-dockerize`
> **Status**: task-ized
> **Created**: 2026-07-03
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-frontend-dockerize/tasks`)
> **Phase inputs**: `explore.md`, `proposal.md`, `specs/*/spec.md`, `design.md`
> **Phase output**: este archivo (referencia para `sdd-apply`)

---

## Resumen para reviewers

- **15 tareas** en 4 fases (Infra → Código → Config → Docs → Verify).
- **Forecast del budget 400-line**: **Medium** (~358 líneas estimadas en `design.md` §12). Single PR.
- **Cadena PR** NO recomendada (el total está dentro del budget y los cambios están cohesivos).
- **Riesgo de review**: 14 archivos, 6 nuevos. La superficie de código de aplicación es mínima (2 rutas + 1 spec + 1 config Playwright + 1 vite.config); el resto es config/infra/docs.
- **Strict TDD**: aplica al componente Go (sin cambios en este PR). Para el frontend, Vitest + Playwright existentes son el corredor; el refactor de las dos rutas se valida con `pnpm test:ci` antes y después (no se requieren specs nuevos).

**Tareas que DEBEN validarse en orden** (no skip):

1. Las **infra** (T1–T5) producen una imagen funcional del frontend antes de tocar código.
2. El **refactor de las dos rutas** (T6–T7) requiere que el frontend esté buildeable (T1–T5 listos).
3. El **e2e** (T8–T9) corre contra el stack dockerizado — depende de T10–T11 (compose).
4. El **verify** (T15) valida todo end-to-end. Es la última tarea y el gate de aceptación.

---

## Fase 1 — Infra del frontend (imagen + nginx)

### T1. Crear `frontend/Dockerfile` multi-stage

- **Output**: `frontend/Dockerfile` (NUEVO, ~50 líneas).
- **Contenido**: builder `node:20-alpine` con corepack + pnpm install + pnpm build; runner `nginx:1.27-alpine` con copia de `dist/` y `nginx.conf`. Build arg `PUBLIC_API_BASE_URL`. Healthcheck wget. CMD nginx foreground.
- **Acceptance criteria**:
  - `docker build -f frontend/Dockerfile ./frontend -t cachicamas/frontend:ci` exit 0.
  - `docker run --rm cachicamas/frontend:ci which node` exit 1 (sin Node en el runner).
  - `docker run --rm cachicamas/frontend:ci ls /usr/share/nginx/html/index.html` exit 0.
  - `docker images cachicamas/frontend:ci --format "{{.Size}}"` reporta < 50MB.
- **Depende de**: ninguna.
- **Spec reference**: `frontend-runtime/spec.md` §"Multi-stage Docker build".

### T2. Crear `frontend/.dockerignore`

- **Output**: `frontend/.dockerignore` (NUEVO, ~15 líneas).
- **Contenido**: `node_modules`, `dist`, `test-results`, `tmp`, `e2e/test-results`, `.atl`, `.git`, `.DS_Store`, `*.log`, `playwright-report`.
- **Acceptance criteria**: el contexto del `docker build` no incluye `node_modules` (verificable con `docker build --no-cache -f frontend/Dockerfile ./frontend 2>&1 | grep "node_modules"` → no debe aparecer).
- **Depende de**: ninguna (pero se valida con T1).

### T3. Crear `frontend/nginx.conf`

- **Output**: `frontend/nginx.conf` (NUEVO, ~75 líneas).
- **Contenido**: server block con `listen 80`, gzip, `/build/` cache 1y, otros assets cache 30d, `/api/` reverse proxy strip-prefix, `/` SPA fallback con `try_files $uri $uri/ /index.html`, `/index.html` no-cache.
- **Acceptance criteria**:
  - `docker run --rm -v $(pwd)/frontend/nginx.conf:/etc/nginx/conf.d/default.conf cachicamas/frontend:ci nginx -t` exit 0 (config válida).
  - (Verificación end-to-end en T15.)
- **Depende de**: T1.
- **Spec reference**: `frontend-runtime/spec.md` §"SPA fallback", §"Reverse proxy", §"Gzip", §"Cache headers".

### T4. Configurar el static adapter de Qwik

- **Output 1**: `frontend/adapters/static/vite.config.ts` (NUEVO, ~25 líneas).
- **Output 2**: `frontend/vite.config.ts` (MODIFIED, +5 líneas).
- **Contenido del adapter**: `staticAdapter({ origin: "http://localhost:3015" })` extendido del base config.
- **Patch en `vite.config.ts`**: registrar el adapter cuando `command === "build"`.
- **Acceptance criteria**:
  - `cd frontend && pnpm build` exit 0.
  - `ls frontend/dist/index.html` existe.
  - `ls frontend/dist/organizations/index.html` existe.
  - `ls frontend/dist/organizations/new/index.html` existe.
  - `ls frontend/dist/organizations/[id]/index.html` NO existe (ruta dinámica, no prerenderizada).
- **Depende de**: T1 (necesita el Dockerfile para probar).
- **Spec reference**: `frontend-runtime/spec.md` §"Static prerender".

### T5. Validar la imagen construida standalone

- **Output**: ninguno (solo verificación).
- **Acceptance criteria**:
  - `docker build -f frontend/Dockerfile ./frontend -t cachicamas/frontend:local` exit 0.
  - `docker run --rm -d -p 8081:80 --name frontend-test cachicamas/frontend:local` exit 0.
  - `sleep 5 && curl -fsS http://localhost:8081/` retorna 200 con HTML del landing.
  - `curl -fsS http://localhost:8081/organizations/999` retorna 200 (SPA fallback).
  - `curl -fsS http://localhost:8081/build/` retorna 404 (directorio no navega; los chunks sí).
  - `docker rm -f frontend-test`.
- **Depende de**: T1, T2, T3, T4.
- **Spec reference**: `frontend-runtime/spec.md` (acepta múltiples requisitos).

---

## Fase 2 — Refactor del código de la app

### T6. Refactor `frontend/src/routes/organizations/index.tsx`

- **Output**: `frontend/src/routes/organizations/index.tsx` (MODIFIED, ~10 net change).
- **Cambio**: `routeLoader$` → `useVisibleTask$` + `useSignal`. Remover import de `routeLoader$`. Mantener el bloque de error con `data-organization-list-error`. Componente `OrganizationList` sin cambios.
- **Acceptance criteria**:
  - `cd frontend && pnpm test:ci` exit 0 (Vitest; el spec renderiza el componente directo, no el loader).
  - `cd frontend && pnpm build.types` exit 0 (no errores de TypeScript).
  - (Verificación end-to-end en T15 con el compose.)
- **Depende de**: T4 (vite.config listo).
- **Spec reference**: `frontend-e2e-and-client-data/spec.md` §"Organizations list loads its data in the browser".

### T7. Refactor `frontend/src/routes/organizations/[id]/index.tsx`

- **Output**: `frontend/src/routes/organizations/[id]/index.tsx` (MODIFIED, ~20 net change).
- **Cambio**: `routeLoader$` → `useVisibleTask$` + `useSignal` + `useLocation()`. Validación client-side de `id > 0`. Manejo de error con `data-organization-error` y "Back to organizations" link. Componente `OrganizationReadback` sin cambios.
- **Acceptance criteria**:
  - `cd frontend && pnpm test:ci` exit 0.
  - `pnpm build.types` exit 0.
  - Manual smoke: navega a `/organizations/1` con el dev server + DB con datos → renderiza el readback.
- **Depende de**: T4, T6 (refactor de la otra ruta antes para validar consistencia).
- **Spec reference**: `frontend-e2e-and-client-data/spec.md` §"Organizations readback loads its data in the browser".

---

## Fase 3 — Configuración de testing

### T8. Refactor `frontend/playwright.config.ts` a env-driven

- **Output**: `frontend/playwright.config.ts` (MODIFIED, ~10 net change).
- **Cambio**: `baseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173"`; `webServer` condicional (`useDevServer ? {...} : undefined`).
- **Acceptance criteria** (RED — antes de T9):
  - `cd frontend && pnpm test:e2e` SIN `E2E_BASE_URL` debe seguir funcionando como antes (levanta dev server, base 5173).
  - `cd frontend && E2E_BASE_URL=http://localhost:9999 pnpm test:e2e` (sin stack en 9999) debe fallar con un error de conexión claro, NO con un timeout de webServer.
- **Depende de**: ninguna (cambio aislado).
- **Spec reference**: `frontend-e2e-and-client-data/spec.md` §"Playwright e2e runner supports two modes".

### T9. Patch `frontend/e2e/create-organization.spec.ts` para esperar el fetch client-side

- **Output**: `frontend/e2e/create-organization.spec.ts` (MODIFIED, +3 líneas).
- **Cambio**: agregar `await page.waitForLoadState("networkidle");` después de `await expect(page).toHaveURL(/\/organizations\/\d+/);`.
- **Acceptance criteria** (GREEN):
  - Con `E2E_BASE_URL` apuntando al compose, el spec pasa el 100% del tiempo (3 corridas consecutivas, sin flake).
  - Sin `E2E_BASE_URL`, el spec sigue pasando en dev local.
- **Depende de**: T8 (sin T8, el cambio no se puede probar a fondo).
- **Spec reference**: `frontend-e2e-and-client-data/spec.md` §"e2e spec waits for client-side data on navigation to readback".

---

## Fase 4 — Compose + .env + Docs

### T10. Servicio `frontend` en `docker-compose.yaml`

- **Output**: `docker-compose.yaml` (MODIFIED, +30 líneas en el bloque `services:`).
- **Cambio**: agregar el servicio `frontend` con `build.context: ./frontend`, `args.PUBLIC_API_BASE_URL: ${PUBLIC_API_BASE_URL:-/api}`, `ports: ["${FRONTEND_PORT:-3015}:80"]`, `networks: [cachicamas_network]`, `depends_on: database_administrator: { condition: service_started }`, `healthcheck: wget --spider`.
- **Acceptance criteria**:
  - `docker compose config` exit 0.
  - El rendered config contiene el servicio `frontend` con las env vars correctas.
- **Depende de**: ninguna (cambio aislado al YAML).
- **Spec reference**: `frontend-runtime/spec.md` §"Healthcheck".

### T11. Crear `docker-compose.vps.yaml` (override file)

- **Output**: `docker-compose.vps.yaml` (NUEVO, ~70 líneas).
- **Cambio**: redefinir `postgres`, `jaeger`, `otel-collector`, `database_administrator` con `ports: []`. En `database_administrator.environment`, re-listar todas las env vars + `SERVICE_ENV: production` + `CORS_ALLOW_ORIGINS: ${CORS_ALLOW_ORIGINS:-http://localhost:3015,https://cachicamas.example.com}`. Comment al tope apuntando al base.
- **Acceptance criteria**:
  - `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config` exit 0.
  - En el rendered config:
    - `frontend.ports` tiene UN entry (`${FRONTEND_PORT:-3015}:80`).
    - `postgres.ports` está ausente (o `[]`).
    - `jaeger.ports` está ausente.
    - `otel-collector.ports` está ausente.
    - `database_administrator.ports` está ausente.
    - `database_administrator.environment.SERVICE_ENV == "production"`.
    - `database_administrator.environment.CORS_ALLOW_ORIGINS` contiene `http://localhost:3015`.
- **Depende de**: T10 (el servicio `frontend` debe existir en el base).
- **Spec reference**: `frontend-compose-and-cors/spec.md` §"Compose override file", §"Override sets production service env".

### T12. Actualizar `.env.example`

- **Output**: `.env.example` (MODIFIED, +15 líneas).
- **Cambio**: agregar `FRONTEND_PORT=3015`, `PUBLIC_API_BASE_URL=/api`, `CORS_ALLOW_ORIGINS=http://localhost:3015,https://cachicamas.example.com` en una nueva sección "Frontend" arriba del bloque de PostgreSQL.
- **Acceptance criteria**:
  - `grep -E '^(FRONTEND_PORT|PUBLIC_API_BASE_URL|CORS_ALLOW_ORIGINS)=' .env.example` retorna las 3 líneas.
- **Depende de**: ninguna.
- **Spec reference**: `frontend-compose-and-cors/spec.md` (implícito — variables documentadas).

### T13. Documentar el deploy en VPS en `README.md` (raíz)

- **Output**: `README.md` (MODIFIED, +20 líneas).
- **Cambio**: agregar sección "Deploy to VPS" después de "Quick start" con los dos comandos (`docker compose up` para dev, `docker compose -f ... -f ...vps.yaml up` para VPS) y una nota sobre `CORS_ALLOW_ORIGINS`.
- **Acceptance criteria**:
  - `grep -A 20 "## Deploy to VPS" README.md` retorna el bloque correcto.
  - El comando para VPS usa `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build`.
- **Depende de**: T11 (el override file debe existir antes de documentarlo).
- **Spec reference**: ninguno específico (es doc).

### T14. Documentar e2e dual mode en `frontend/README.md`

- **Output**: `frontend/README.md` (MODIFIED, +10 líneas en la sección "End-to-end tests").
- **Cambio**: agregar los 3 modos de `E2E_BASE_URL` con ejemplos copy-pasteables.
- **Acceptance criteria**:
  - `grep -B 2 -A 15 "E2E_BASE_URL" frontend/README.md` retorna los 3 modos.
- **Depende de**: T9 (el behavior debe estar funcionando antes de documentarlo).

---

## Fase 5 — Verificación end-to-end

### T15. Validar el stack dockerizado completo

- **Output**: resultado del verify phase (no es un archivo, es la ejecución).
- **Acceptance criteria** (todos deben pasar):
  - [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config` exit 0.
  - [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build` levanta los 5 servicios.
  - [ ] `docker compose ps` reporta `frontend` como `(healthy)` en < 30s.
  - [ ] `docker images cachicamas/frontend:local --format "{{.Size}}"` reporta < 50MB.
  - [ ] `curl -fsS http://localhost:3015/` retorna 200 con HTML del brand.
  - [ ] `curl -fsS http://localhost:3015/organizations` retorna 200 (no 404, prerender OK).
  - [ ] `curl -fsS http://localhost:3015/organizations/999999` retorna 200 (SPA fallback OK).
  - [ ] `curl -fsS -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "full_name=Verify&identification=verify-$(date +%s)" http://localhost:3015/api/organizations` retorna 201.
  - [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml exec database_administrator wget -q -O- http://localhost:8080/health` retorna `{"status":"ok"}`.
  - [ ] `cd frontend && pnpm test:e2e` SIN env var (dev local) sigue pasando.
  - [ ] `cd frontend && E2E_BASE_URL=http://localhost:3015 pnpm test:e2e` pasa contra el compose (3 corridas consecutivas, sin flake).
  - [ ] `cd frontend && pnpm test:ci` sigue verde.
  - [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml down -v` cleanup OK.
- **Si algo falla**: parar, reportar el fallo con la salida exacta del comando, NO continuar con apply/archive. El parent decide si retry / fix / abort.
- **Depende de**: T1–T14 todas.
- **Spec reference**: TODOS los specs (es la verificación global).

---

## Cambios de configuración — resumen

| Archivo | Tipo | Diff size (líneas) | Spec touched |
| --------- | ------ | --------------------- | -------------- |
| `frontend/Dockerfile` | New | +50 | frontend-runtime |
| `frontend/.dockerignore` | New | +15 | frontend-runtime |
| `frontend/nginx.conf` | New | +75 | frontend-runtime |
| `frontend/adapters/static/vite.config.ts` | New | +25 | frontend-runtime |
| `frontend/vite.config.ts` | Modified | +5 | frontend-runtime |
| `frontend/src/routes/organizations/index.tsx` | Modified | ~10 net | frontend-e2e-and-client-data |
| `frontend/src/routes/organizations/[id]/index.tsx` | Modified | ~20 net | frontend-e2e-and-client-data |
| `frontend/playwright.config.ts` | Modified | ~10 net | frontend-e2e-and-client-data |
| `frontend/e2e/create-organization.spec.ts` | Modified | +3 | frontend-e2e-and-client-data |
| `docker-compose.yaml` | Modified | +30 | frontend-compose-and-cors |
| `docker-compose.vps.yaml` | New | +70 | frontend-compose-and-cors |
| `.env.example` | Modified | +15 | frontend-compose-and-cors |
| `README.md` | Modified | +20 | (docs) |
| `frontend/README.md` | Modified | +10 | (docs) |
| **TOTAL** | — | **~358 líneas** | — |

---

## Forecast de review workload

- **Total diff**: ~358 líneas (single PR).
- **400-line budget**: 89% consumido.
- **Riesgo de review**: Medium.
- **Chained PR NO recomendado** (dentro del budget y cohesivo).

**Recomendación para el reviewer**: auditar en este orden:

1. `design.md` §3, §4, §6 (los cambios no triviales: Dockerfile, nginx, refactor de rutas).
2. `docker-compose.vps.yaml` (override — cambia el contrato de deployment).
3. `cors.go` (verificar que NO se tocó — `git diff backend/` debe ser 0 líneas).
4. El resto de los cambios es config/docs/specs y se valida en menos de 10 min.

**Bloqueadores automáticos** (deben ser 0 líneas de diff):

- `backend/database_administrator/src/**` — el Go bin no se toca en este change.
- `infra/**` — el infra del repo no se toca en este change.
- `openspec/specs/**` (los specs viejos en `openspec/specs/db-migrations/spec.md` y otros) — los specs viejos no se tocan; los nuevos van en `openspec/changes/{change}/specs/`.

---

## Plan de ejecución

```
T1 ──► T2 ──► T3 ──► T4 ──► T5 ──► T6 ──► T7 ──► T8 ──► T9
                                   │               │
                                   │               └─► T10 ──► T11 ──► T12 ──► T13 ──► T14
                                   │                                                     │
                                   └──────────────────────────────────────────────────► T15
```

Notas:

- T5 es un gate de calidad (la imagen standalone debe correr antes de tocar código de app).
- T8 es RED antes de T9 (el cambio del config primero, el patch del spec después para validar el GREEN).
- T15 es el verify global; requiere todo lo anterior.

---

## Strict TDD notes

- **T1–T5 (infra)**: no requieren tests nuevos (es Docker + nginx). El "test" es la imagen construyéndose + respondiendo a curl. No hay framework de test para Docker configs en este repo.
- **T6, T7 (refactor rutas)**: Vitest spec del componente presentacional (`OrganizationList`, `OrganizationReadback`) **NO requiere cambios** porque renderiza el componente directo, no la ruta refactorizada. La regla de strict TDD se honra así: el comportamiento que cambia (loader → signal) está cubierto por el e2e (T15). El Vitest verifica que el componente presentacional sigue funcionando aislado. **Si strict TDD exigiera un Vitest específico para el nuevo path**, agregar `frontend/src/routes/organizations/index.signal.spec.tsx` que mockee `listOrganizations` y verifique el signal. **Decisión**: no agregar (el e2e es la cobertura; agregar un Vitest específico duplica cobertura). Documentar.
- **T8 (playwright config)**: aplicar RED-first. Cambiar el config primero; verificar el comportamiento nuevo (con `E2E_BASE_URL`) ANTES de ajustar el spec (T9).
- **T9 (spec patch)**: aplicar GREEN. El patch es chico (+3 líneas) y se verifica con `E2E_BASE_URL=... pnpm test:e2e`.
- **T10–T14 (config/docs)**: no requieren tests (no hay framework para esto).
- **T15 (verify)**: el verify phase corre los tests existentes + nuevos asserts y genera el `verify-report.md`.

---

## Resultado

- **Status**: ok
- **Executive summary**: 15 tareas en 5 fases. Single PR, ~358 líneas, Medium risk. T15 (verify) es el gate final y valida el stack dockerizado end-to-end con Playwright apuntando a `:3015`.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/tasks.md` (este archivo), engram topic `sdd/cachicamas-frontend-dockerize/tasks`.
- **Next recommended**: `sdd-apply` — ejecutar las tareas T1–T15 en orden, gate en T15 (verify).
- **Skill resolution**: `paths-injected`.
