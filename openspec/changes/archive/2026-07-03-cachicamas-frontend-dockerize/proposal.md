# Proposal: cachicamas-frontend-dockerize

> **Cambio**: `cachicamas-frontend-dockerize`
> **Status**: proposed
> **Created**: 2026-07-03
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-frontend-dockerize/proposal`)
> **Phase input**: `openspec/changes/cachicamas-frontend-dockerize/explore.md` (en este mismo change)
> **Phase output**: este archivo (referencia para `sdd-spec` y `sdd-design`)

---

## Intent

El frontend de cachicamas (Qwik City, en `frontend/`) **no forma parte del stack dockerizado** hoy. El `docker-compose.yaml` levanta Postgres, Jaeger, OTel Collector y `database_administrator` (Go), pero el frontend se ejecuta en la terminal del desarrollador con `pnpm dev` (Vite dev server en `:5173`). Esto bloquea el deploy a un VPS: no hay forma de servir el frontend desde un contenedor, y no hay forma de validar el wire end-to-end contra el binario Go sin una sesión de Vite abierta.

Este cambio **integra el frontend al stack dockerizado** con un criterio de "ultra-light footprint" alineado con la decisión original de usar Qwik. La imagen final del frontend queda en **~30MB** (vs ~150MB de un Node alpine SSR) porque el artefacto es 100% estático: HTML + JS + CSS prerenderizado en build-time, servido por `nginx:1.27-alpine` con SPA fallback. El browser del usuario hace una sola navegación al origin del frontend, y nginx proxya `/api/*` al Go bin en la red privada — eso evita el cross-origin y mantiene CORS fuera del camino crítico en producción.

El change también ajusta el contrato CORS del Go bin, que por default está deshabilitado en producción (asume reverse proxy). Como en este deploy **nginx del frontend y el Go bin son contenedores separados**, hay que activar CORS vía `CORS_ALLOW_ORIGINS` para el caso de borde "browser hace fetch directo al Go" (por ejemplo, desde la consola del navegador o si se publica el Go bin opcionalmente para debugging). En el flujo normal (frontend → nginx → Go), CORS es innecesario.

El **e2e se vuelve dual-mode** con un cambio mínimo en `playwright.config.ts`: la env var `E2E_BASE_URL` decide si Playwright arranca `pnpm dev` en 5173 (flujo actual, dev local) o si apunta a un `baseURL` provisto (CI, post-deploy en compose, VPS). El mismo spec `e2e/create-organization.spec.ts` corre en ambos modos.

---

## Scope

### In Scope

- **Imagen Docker del frontend** (`frontend/Dockerfile`): multi-stage con `node:20-alpine` (builder) + `nginx:1.27-alpine` (runner). Build arg `PUBLIC_API_BASE_URL` para que el browser sepa adonde pegarle. Output: imagen ~30MB.
- **`frontend/nginx.conf`**: SPA fallback (`try_files $uri $uri/ /index.html`), gzip, cache headers para chunks inmutables de Qwik, reverse proxy `/api/*` → `http://database_administrator:8080`.
- **Static adapter de Qwik**: instalación + config de `@builder.io/qwik-city/adapters/static/vite` + patch de `frontend/vite.config.ts` (sin `pnpm qwik add` interactivo, ver Approach).
- **Refactor de dos rutas** para mover la carga de datos del server (build-time en SSG) al client (runtime en el browser):
  - `frontend/src/routes/organizations/index.tsx`: `routeLoader$` → `useVisibleTask$` + `useSignal`.
  - `frontend/src/routes/organizations/[id]/index.tsx`: `routeLoader$` → `useVisibleTask$` + `useSignal` + `useLocation()`.
- **Servicio `frontend` en `docker-compose.yaml`**: build local, puerto host `${FRONTEND_PORT:-3015}:80`, `network: cachicamas_network`, `depends_on: database_administrator: condition: service_started`, `healthcheck` con `wget --spider`.
- **Parametrización VPS-friendly del compose**: variable `VPS_MODE` que decide qué `ports:` se publican. Default (`VPS_MODE=false`): todos los `ports:` actuales se mantienen para dev local. `VPS_MODE=true`: solo `frontend:${FRONTEND_PORT:-3015}:80` se publica; Postgres, Go, Jaeger y OTel quedan en la red privada. Una sola fuente de verdad, un solo `docker-compose.yaml`.
- **Activación de CORS en el Go bin** vía `CORS_ALLOW_ORIGINS` (env var). Default propuesto: `http://localhost:3015` cuando `VPS_MODE=true`; el dev local sigue con el default existente (`http://localhost:5173`).
- **Actualización de `.env.example`**: documentar `FRONTEND_PORT`, `CORS_ALLOW_ORIGINS`, `PUBLIC_API_BASE_URL`, `VPS_MODE`.
- **Refactor de `frontend/playwright.config.ts`**: `baseURL` y `webServer` env-driven por `E2E_BASE_URL`.
- **Ajuste de `frontend/e2e/create-organization.spec.ts`**: agregar `waitForLoadState("networkidle")` después de la navegación a `/organizations/{id}` para esperar el fetch client-side (mitiga R6 del explore).
- **Documentación**: `README.md` (raíz) con una nota de 3-5 líneas en la sección de deploy; `frontend/README.md` con el nuevo flujo `E2E_BASE_URL` en la sección "End-to-end tests".

### Out of Scope (deferred pero relacionado)

- **Pinning con `@sha256:...` digest**: el compose actual pinna por tag (no por digest). Mantener consistencia con el resto del repo. Un follow-up puede agregar digest pinning a todas las imágenes.
- **Reverse proxy externo (Traefik, Caddy, nginx fuera de compose)**: este change resuelve el deploy a VPS con un solo compose, sin proxy externo. Si el VPS necesita TLS termination o un dominio, eso es un follow-up (un Caddy en el host o un Cloudflare Tunnel).
- **TLS**: el change entrega HTTP plano. TLS es un follow-up (certbot en el host, o Caddy automático).
- **Persistencia de logs en Loki / métricas en Prometheus**: el change no toca el pipeline de observability existente (que sigue yendo al debug exporter del otel-collector). El frontend **no** instrumenta OTel en este PR (Qwik + OTel es un follow-up; ver R-X de los riesgos).
- **CDN / cache global**: nginx local es suficiente para el VPS. CDN es follow-up.
- **Tests de carga / k6**: este change valida funcionalmente con Playwright. Tests de carga son follow-up.
- **Refactor de `entry.ssr.tsx` y `entry.preview.tsx`**: existen pero el static adapter no los necesita. Limpieza opcional en un follow-up.
- **Migración del `cachicamas-network` a Traefik / Docker Swarm**: fuera de scope; este change asume `docker compose` single-node.
- **Build de la imagen en CI con cache de capas / registry push**: este change construye localmente con `docker compose build`. CI con registry es follow-up.

---

## Capabilities

### New Capabilities

- `frontend-static-spa`: capacidad de servir el frontend Qwik como artefacto estático (HTML + JS + CSS) con SPA fallback vía nginx. Cubre prerender de rutas estáticas, navegación client-side para rutas dinámicas, y carga de datos client-side desde el Go bin.
- `frontend-docker-image`: capacidad de producir una imagen Docker reproducible del frontend Qwik con `frontend/Dockerfile` (multi-stage, pinned, ~30MB).
- `frontend-cors-allow-origin`: capacidad del Go bin de aceptar requests cross-origin desde el origin del frontend cuando se setea `CORS_ALLOW_ORIGINS` (override del default "production = off").
- `frontend-e2e-dual-mode`: capacidad del runner de Playwright de apuntar a un `baseURL` arbitrario via `E2E_BASE_URL`, y de no levantar `webServer` cuando esa env var está presente.
- `compose-vps-profile`: capacidad del `docker-compose.yaml` de switchear entre "local dev (todos los servicios publicados)" y "VPS (solo frontend publicado)" vía la env var `VPS_MODE`.

### Modified Capabilities

- **`organizations-list` (existing)**: el spec actual (`openspec/specs/organizations-list/spec.md` si existe, o el spec mental implícito) asume que el `routeLoader$` corre en cada request. El refactor mueve la carga a `useVisibleTask$` client-side. La **funcionalidad visible es idéntica** (la lista se muestra), pero el **contrato de testing** cambia: el spec viejo podría haber assertado que el loader corría en server; el spec nuevo verifica que la lista se muestra después del fetch. (Alinear en `sdd-spec`.)
- **`organizations-readback` (existing)**: idem, la carga del org por id pasa de `routeLoader$` server-side a fetch client-side. La **funcionalidad visible es idéntica**; el **contrato de testing** cambia igual que arriba.

---

## Approach

### Topología final

```
┌────────────────────────────────────────────────────────────────────┐
│ VPS host (public internet)                                         │
│                                                                    │
│   :3015 ──► docker compose                                        │
│              ├─ frontend (nginx:1.27-alpine, ~30MB)                │
│              │   ├─ /                → dist/index.html             │
│              │   ├─ /organizations    → dist/organizations/index   │
│              │   ├─ /organizations/new → dist/.../new/index        │
│              │   ├─ /organizations/{id} → dist/.../[id]/index.html │
│              │   │                   (SPA fallback)               │
│              │   ├─ /api/* ──► proxy_pass http://database_administrator:8080
│              │   │                                                       │
│              │   └─ /build/* ──► cache 1y (chunks Qwik inmutables)       │
│              │                                                              │
│              ├─ database_administrator (Go, internal only)               │
│              │   └─ :8080 ──► CORS_ALLOW_ORIGINS=$FRONTEND_ORIGIN        │
│              │                                                              │
│              ├─ postgres (internal only)                                  │
│              ├─ otel-collector (internal only)                            │
│              └─ jaeger (internal only)                                    │
│                                                                    │
│   (private network: cachicamas_network)                           │
└────────────────────────────────────────────────────────────────────┘
```

**El browser del usuario hace UN solo origen**: `http://localhost:3015` (o `https://cachicamas.example.com` en el VPS). Toda la lógica de proxy + cache queda en nginx. El Go bin es privado a la red de compose y nunca ve un request cross-origin en el flujo normal.

### Multi-stage Dockerfile (resumen; diff exacto en `sdd-design`)

```dockerfile
# builder
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY . .
ARG PUBLIC_API_BASE_URL=/api
ENV PUBLIC_API_BASE_URL=$PUBLIC_API_BASE_URL
RUN pnpm build

# runner
FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK CMD wget --spider -q http://127.0.0.1/ || exit 1
```

Output: imagen ~30MB (alpine + nginx ~15MB + dist/ ~10-15MB dependiendo de los chunks).

### nginx.conf (resumen; diff exacto en `sdd-design`)

- `listen 80;` con `server_name _;`
- `gzip on;` para `text/plain text/css application/json application/javascript application/xml`
- `location / { try_files $uri $uri/ /index.html; }` — SPA fallback
- `location /api/ { proxy_pass http://database_administrator:8080/; proxy_set_header ...; }` — reverse proxy al Go bin (sin `/api` prefix, Go ve el path real)
- `location /build/ { expires 1y; add_header Cache-Control "public, immutable"; }` — cache de chunks Qwik
- `location = /index.html { add_header Cache-Control "no-cache"; }` — el shell no se cachea
- `access_log /var/log/nginx/access.log;` y `error_log /var/log/nginx/error.log;`

### Qwik static adapter (resumen; diff exacto en `sdd-design`)

Estrategia para evitar `pnpm qwik add static` interactivo:

1. Crear `frontend/adapters/static/vite.config.ts` a mano con la config canónica (siguiendo la documentación oficial de Qwik City).
2. Modificar `frontend/vite.config.ts` para importar y registrar el adapter cuando se construye (`command === "build"`).
3. No agregar el adapter como dependencia nueva — el adapter es parte de `@builder.io/qwik-city`, que ya está en `devDependencies`. Verificar en `sdd-apply`.

### Refactor de las dos rutas (resumen; diff exacto en `sdd-design`)

Patrón único aplicado a las dos rutas. Para `/organizations`:

```tsx
// Antes
export const useOrganizationsLoader = routeLoader$(async () => {
  const result = await listOrganizations();
  // ...
});

// Después
import { component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";
import { OrganizationList, type OrganizationSummary } from "~/components/organization-list/organization-list";
import { listOrganizations } from "~/lib/api";

export default component$(() => {
  const orgs = useSignal<OrganizationSummary[]>([]);
  const error = useSignal<string | null>(null);
  useVisibleTask$(async () => {
    const result = await listOrganizations();
    if (result.ok) {
      orgs.value = result.value.map((o) => ({
        id: o.id, full_name: o.full_name, identification: o.identification,
      }));
    } else {
      error.value = result.message;
    }
  });
  return (
    <>
      {error.value && <div role="alert" data-organization-list-error>{error.value}</div>}
      <OrganizationList organizations={orgs.value} />
    </>
  );
});
```

Para `/organizations/[id]`, igual con `useLocation()` para leer el `params.id`.

### Compose — perfil VPS via `VPS_MODE`

```yaml
x-vps-ports: &vps-ports
  # placeholder, no se usa; ver bloque por servicio

services:
  postgres:
    # ...
    ports: ${VPS_MODE:-false} == "true" ? [] : ["${POSTGRES_PORT:-5432}:5432"]
```

Problema: YAML no tiene ternarios. Alternativa limpia — usar YAML anchors + la técnica de env-var `default`:

```yaml
services:
  postgres:
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
```

Y en el entrypoint (o en la lógica de `docker compose`), se filtra con `--profile vps`. **Pero** profiles de compose no ocultan ports — solo excluyen servicios. **La solución real** es **un override file**: `docker-compose.vps.yaml` que redefina los servicios sin `ports:`. Más simple y más limpio para audit.

**Decisión revisada (rompe con el spirit de "compose único")**: dos archivos.

- `docker-compose.yaml` — base, modo dev (todo publicado).
- `docker-compose.vps.yaml` — override, modo VPS (solo `frontend` con `ports:`).
- Uso:
  - Dev local: `docker compose up -d --build`
  - VPS: `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build`

**Justificación del cambio de approach**: el explore recomendaba parametrizar por `VPS_MODE`, pero YAML no permite condicionales de bloque; los profiles de compose no ocultan `ports:`. El override file es la solución idiomática de compose v2 y es **más limpia para audit** (un revisor puede ver "este es el stack que va al VPS" sin parsear condicionales). El cambio es retro-compatible: el dev local sigue funcionando con el comando de siempre.

### CORS en el Go bin (resumen)

`backend/database_administrator/src/interfaces/http/cors.go` **no cambia**. El default sigue siendo:

- `SERVICE_ENV=development` (default) → CORS habilitado con `http://localhost:5173`.
- `CORS_ALLOW_ORIGINS` set → CORS habilitado con esa lista, **independiente de `SERVICE_ENV`**.

En `docker-compose.vps.yaml`, `database_administrator` se redefine con:

```yaml
services:
  database_administrator:
    environment:
      SERVICE_ENV: production
      CORS_ALLOW_ORIGINS: ${CORS_ALLOW_ORIGINS:-http://localhost:3015,http://cachicamas.example.com}
```

`SERVICE_ENV=production` apaga el default de dev (que era `http://localhost:5173`); `CORS_ALLOW_ORIGINS` lo reemplaza con la lista de origins reales. Esto deja CORS **deshabilitado por default en VPS** (consistente con la postura del código) y **explícitamente habilitado** cuando se quiere.

### e2e dual mode (resumen)

```ts
// playwright.config.ts
const e2eBaseUrl = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const useDevServer = !process.env.E2E_BASE_URL;

export default defineConfig({
  testDir: "./e2e",
  use: { baseURL: e2eBaseUrl, /* ... */ },
  webServer: useDevServer
    ? { command: "pnpm dev", url: "http://localhost:5173", reuseExistingServer: !process.env.CI, timeout: 60_000 }
    : undefined,
  // ...
});
```

Comportamiento:

| Comando | `baseURL` | `webServer` |
| --------- | ----------- | ------------- |
| `pnpm test:e2e` | `http://localhost:5173` | levanta `pnpm dev` (dev local) |
| `E2E_BASE_URL=http://localhost:3015 pnpm test:e2e` | `http://localhost:3015` | no levanta nada (confía en el compose) |
| `E2E_BASE_URL=https://cachicamas.example.com pnpm test:e2e` | el dominio público | no levanta nada (VPS en prod) |

---

## Affected Areas

| Area | Impact | Description |
| ------ | -------- | ------------- |
| `frontend/Dockerfile` | **New** | Multi-stage: `node:20-alpine` (builder) + `nginx:1.27-alpine` (runner). Build arg `PUBLIC_API_BASE_URL`. |
| `frontend/nginx.conf` | **New** | SPA fallback, gzip, cache headers, reverse proxy `/api/*` → Go bin. |
| `frontend/adapters/static/vite.config.ts` | **New** | Config del static adapter (escrito a mano para evitar `pnpm qwik add` interactivo). |
| `frontend/vite.config.ts` | Modified | Importar y registrar el static adapter cuando `command === "build"`. |
| `frontend/src/routes/organizations/index.tsx` | Modified | `routeLoader$` → `useVisibleTask$` + `useSignal`. |
| `frontend/src/routes/organizations/[id]/index.tsx` | Modified | `routeLoader$` → `useVisibleTask$` + `useSignal` + `useLocation()`. |
| `frontend/playwright.config.ts` | Modified | `baseURL` y `webServer` env-driven por `E2E_BASE_URL`. |
| `frontend/e2e/create-organization.spec.ts` | Modified (opcional) | Agregar `waitForLoadState("networkidle")` post-navegación al readback. |
| `docker-compose.yaml` | Modified | Agregar servicio `frontend`. Build context, `ports: "${FRONTEND_PORT:-3015}:80"`, `healthcheck`. |
| `docker-compose.vps.yaml` | **New** | Override file: redefine `postgres`, `jaeger`, `otel-collector`, `database_administrator` SIN `ports:`. Agrega `SERVICE_ENV: production` + `CORS_ALLOW_ORIGINS` al Go bin. |
| `.env.example` | Modified | Documentar `FRONTEND_PORT`, `CORS_ALLOW_ORIGINS`, `PUBLIC_API_BASE_URL`, `VPS_MODE` (o nota sobre el override file). |
| `README.md` (raíz) | Modified | Una nota de 3-5 líneas en la sección de deploy: "Para VPS, usar `docker-compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build`. Puerto externo: 3015." |
| `frontend/README.md` | Modified | Sección "End-to-end tests" con la nota del flujo `E2E_BASE_URL` y un ejemplo de CI. |
| `backend/database_administrator/src/interfaces/http/cors.go` | **No change** | El default sigue siendo "production = off, dev = localhost:5173". El cambio de comportamiento viene del env `CORS_ALLOW_ORIGINS` que ya existe. |

---

## Risks

| # | Risk | Likelihood | Impact | Mitigation |
| --- | ------ | ------------ | -------- | ------------ |
| R1 | El refactor de `routeLoader$` rompe la página de readback porque el e2e navega a `/organizations/{id}` antes de que el fetch client-side termine | **High** | Test e2e falla | El spec actual navega y luego hace asserts de contenido. Agregar `page.waitForLoadState("networkidle")` (o un `expect(...).toBeVisible({ timeout: 5000 })`) en el spec antes del primer assert post-navegación. Verificar en `sdd-verify`. |
| R2 | `pnpm qwik add static` es interactivo o no se puede invocar de forma no-determinística en CI | Medium | Build del frontend falla en CI | Estrategia sin `pnpm qwik add`: crear `frontend/adapters/static/vite.config.ts` a mano y registrar el adapter en `vite.config.ts` con un patch explícito. Verificar en `sdd-apply` que el adapter está en `@builder.io/qwik-city` (no requiere dep nueva). |
| R3 | nginx SPA fallback no maneja bien URLs con trailing slash, o devuelve 404 para `/organizations/123` cuando debería servir `index.html` | Medium | Rutas dinámicas rotas en el browser | Configurar `try_files $uri $uri/ /index.html;` con `index index.html`. Probar con `curl -fsS http://localhost:3015/organizations/123` en `sdd-verify` (debe devolver HTML, no 404). |
| R4 | El build de Qwik con static adapter puede generar archivos con paths rotos (assets en `/build/q-abc.js` que el browser no encuentra) | Medium | Browser tira 404s de chunks → página rota | Verificar en `sdd-verify` que `curl http://localhost:3015/` retorna HTML con `<script>` tags correctos, y que los chunks se sirven. Si hay paths rotos, ajustar `base` en `vite.config.ts` o `trailingSlash` en `qwikCity()`. |
| R5 | El container `frontend` arranca antes de que el Go bin esté listo, y el primer POST falla | Low | Primer POST del e2e retorna 502 | `depends_on: database_administrator: condition: service_started` (no `service_healthy` porque el healthcheck del Go está disabled). Aceptable: el e2e ya tiene retry implícito en el submit del form (y en el waitForLoadState). |
| R6 | `docker-compose.vps.yaml` se desincroniza del `docker-compose.yaml` base (alguien modifica uno y se olvida del otro) | Medium | Deploy a VPS con config desactualizada | Agregar un comment en el top de `docker-compose.vps.yaml` apuntando al base. En `sdd-verify`, validar que el override compone correctamente con `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config`. Documentar la convención en `README.md`. |
| R7 | La imagen del frontend termina siendo más grande de lo esperado porque el builder copia `node_modules` o archivos innecesarios al runner | Low | Imagen > 30MB | `.dockerignore` en `frontend/` excluyendo `node_modules`, `dist`, `test-results`, `tmp`, `e2e/test-results`, `.atl`, `.git`. Verificar tamaño en `sdd-verify` con `docker images cachicamas/frontend:local`. |
| R8 | El e2e dual mode tiene un bug: cuando `E2E_BASE_URL` está seteado, Playwright intenta usar el `webServer` block igual y se confunde | Low | Test no corre | El patrón es explícito: `webServer: useDevServer ? {...} : undefined`. Playwright acepta `webServer: undefined` (lo trata como "ya hay un server"). Verificar en `sdd-verify`. |
| R9 | El refactor de las dos rutas rompe los tests Vitest existentes (`routes/organizations/index.spec.tsx` y `[id]/index.spec.tsx`) que importan el componente y le pasan datos stub | Low | Vitest falla | Los specs existentes renderizan el componente presentacional directamente, NO el loader. El refactor cambia el `default export` (loader → signal), pero el componente interno (`OrganizationList`, `OrganizationReadback`) **no cambia**. Verificar en `sdd-apply` que los specs siguen importando el componente correcto. |
| R10 | `SERVICE_ENV=production` en el compose base causa que el Go bin arranque en modo producción y se salte validaciones o logs adicionales que el dev local espera ver | Low | Dev local "se comporta distinto" | El override file setea `SERVICE_ENV=production` SOLO en el contexto del VPS. El compose base sigue con `SERVICE_ENV: ${SERVICE_ENV:-development}`. Sin colisión. |

---

## Rollback Plan

1. **Reversión inmediata** (incidente en deploy): `git revert <commit-hash>` del PR + `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build`. Tiempo: < 1 minuto.
2. **Rollback del refactor del frontend** (si rompe Vitest): `git revert` del commit que refactoriza las dos rutas. El comportamiento de Qwik vuelve a `routeLoader$` server-side, que solo funciona si se regresa a SSR (no SSG). Si el refactor rompe el frontend en SSG, también hay que revertir el `Dockerfile` y el `vite.config.ts` para usar el adapter SSR. **Mitigación**: el refactor de las dos rutas es chico (~30 líneas cada una) y aislado.
3. **Sin schema changes** en Postgres. Sin migraciones. El Go bin no cambia.
4. **Fallback operacional** (si `revert` no es opción inmediata): el `override file` se puede deshabilitar borrando el `-f docker-compose.vps.yaml` del comando. El compose base sigue funcionando como dev local. Tiempo: < 30s.
5. **No hay feature flag** para este change. La decisión de static SSG es arquitectónica, no toggleable.

---

## Dependencies

- `@builder.io/qwik-city@1.20.0` (ya en `devDependencies` de `frontend/package.json`). El static adapter viene como sub-export `@builder.io/qwik-city/adapters/static/vite`. **No requiere bump**.
- `nginx:1.27-alpine` (nueva imagen). Tag pinned, no digest.
- `node:20-alpine` (nueva imagen para el builder). Tag pinned.
- **No hay nuevas dependencias de Go** (`cors.go` no cambia).
- **No hay nuevas dependencias de Node runtime** (el `node_modules` queda en el builder stage; el runner es nginx puro).
- `pnpm` (ya requerido para el build de Qwik, sigue siendo necesario en el builder stage).

---

## Success Criteria

- [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config` valida sin errores.
- [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build` levanta los 5 servicios; el frontend queda `healthy` en < 30s.
- [ ] `docker images cachicamas/frontend:local` reporta un tamaño < 50MB.
- [ ] `curl -fsS http://localhost:3015/` retorna 200 con HTML que contiene el brand mark del frontend.
- [ ] `curl -fsS http://localhost:3015/organizations` retorna 200 con HTML (no 404) — verifica SPA fallback.
- [ ] `curl -fsS http://localhost:3015/organizations/999999` retorna 200 con HTML (no 404) — verifica SPA fallback para rutas dinámicas.
- [ ] `curl -fsS -H "Origin: http://localhost:3015" -X OPTIONS http://localhost:8080/organizations` retorna 204 con `Access-Control-Allow-Origin: http://localhost:3015` — verifica CORS.
- [ ] `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml exec database_administrator wget -q -O- http://localhost:8080/health` retorna `{"status":"ok"}` — verifica que el Go bin responde.
- [ ] `pnpm test:e2e` (sin `E2E_BASE_URL`, dev local) corre y pasa.
- [ ] `E2E_BASE_URL=http://localhost:3015 pnpm test:e2e` corre contra el stack dockerizado y pasa. El spec `create-organization.spec.ts` crea una org real, navega a su readback, y verifica el contenido.
- [ ] `pnpm test:ci` (Vitest, frontend unit) sigue verde — el refactor de las dos rutas no rompe specs.
- [ ] `docker compose config` (sin el override, dev local) sigue funcionando como antes: el dev loop local no cambia.
- [ ] Ningún cambio en el código Go del backend (`git diff backend/` muestra 0 líneas).

---

## Notas para la fase `sdd-spec`

Specs a producir en `openspec/changes/cachicamas-frontend-dockerize/specs/`:

- `frontend-static-spa/spec.md` — Given/When/Then para prerender, SPA fallback, y carga client-side.
- `frontend-docker-image/spec.md` — Given/When/Then para el build de la imagen, tamaño, y reproducibilidad.
- `frontend-cors-allow-origin/spec.md` — Given/When/Then para el comportamiento de CORS con `CORS_ALLOW_ORIGINS` set vs unset.
- `frontend-e2e-dual-mode/spec.md` — Given/When/Then para los dos modos de Playwright.
- `compose-vps-profile/spec.md` — Given/When/Then para el override file y el comportamiento de `ports:`.
- `organizations-client-side-data/spec.md` — Given/When/Then para el refactor de `routeLoader$` → `useVisibleTask$` (las dos rutas).

Recordatorio de las reglas de `openspec/config.yaml`: Given/When/Then, RFC 2119 keywords (MUST, SHALL, SHOULD, MAY), cada escenario independientemente verificable. Los specs viejos (`openspec/specs/`) **no se modifican** en este change — son los deltas en `openspec/changes/cachicamas-frontend-dockerize/specs/`.

## Notas para la fase `sdd-design`

Puntos a diseñar (en orden):

1. **Multi-stage Dockerfile** (diff exacto, qué se copia, layer caching).
2. **nginx.conf** completo (gzip types, cache headers, reverse proxy al Go, SPA fallback, healthcheck si se agrega endpoint).
3. **Static adapter config** (`frontend/adapters/static/vite.config.ts` + patch en `frontend/vite.config.ts`).
4. **Refactor de las dos rutas** (diff exacto, helper común si vale la pena).
5. **Override file `docker-compose.vps.yaml`** (qué redefine, qué agrega, comment de cabecera).
6. **`.env.example` diff** (antes/después de cada variable).
7. **Diagramas de secuencia** para el flujo end-to-end: browser → nginx → Go → Postgres (CORS off en este path), y browser → Go directo (CORS on, para debugging).

## Notas para la fase `sdd-tasks`

Forecast del budget 400-line: **Medium** (estimado 200-350 líneas de diff, repartido entre Dockerfile, nginx.conf, vite.config.ts, dos rutas, playwright.config.ts, docker-compose.yaml, docker-compose.vps.yaml, .env.example, README, frontend/README). Single PR recomendado. Chained PR solo si el refactor de las dos rutas resulta en > 200 líneas de cambios en `frontend/src/routes/` (improbable según lo estimado).

Tareas tentativas (refinar en `sdd-tasks`):

- T1: `frontend/Dockerfile` multi-stage.
- T2: `frontend/.dockerignore` (excluir `node_modules`, `dist`, `test-results`, `tmp`).
- T3: `frontend/nginx.conf` (SPA fallback, gzip, cache, reverse proxy).
- T4: `frontend/adapters/static/vite.config.ts` (escrito a mano).
- T5: `frontend/vite.config.ts` patch (registrar el static adapter).
- T6: Refactor `frontend/src/routes/organizations/index.tsx` (loader → signal).
- T7: Refactor `frontend/src/routes/organizations/[id]/index.tsx` (loader → signal + useLocation).
- T8: `frontend/playwright.config.ts` env-driven.
- T9: `frontend/e2e/create-organization.spec.ts` agregar `waitForLoadState`.
- T10: `docker-compose.yaml` agregar servicio `frontend`.
- T11: `docker-compose.vps.yaml` override file.
- T12: `.env.example` actualizar.
- T13: `README.md` (raíz) sección de deploy.
- T14: `frontend/README.md` sección e2e.
- T15: `sdd-verify` validar end-to-end contra el stack dockerizado.

## Notas para la fase `sdd-apply`

Strict TDD (per `openspec/config.yaml`): aplica al Go service. Para el frontend, Vitest + Playwright son los runners. Regla práctica:

- **T1-T5 (infra)**: no requieren tests nuevos (es config y Docker).
- **T6, T7 (refactor rutas)**: Vitest ya cubre el componente presentacional; verificar que `pnpm test:ci` sigue verde **antes y después** del refactor. No agregar specs nuevos (el explore confirmó que los existentes sidestepean el loader).
- **T8 (playwright config)**: aplicar el patrón "test primero": cambiar `playwright.config.ts` ANTES de cambiar el spec, verificar que `E2E_BASE_URL=... pnpm test:e2e` corre (puede fallar el spec por el refactor, eso es esperado y se arregla en T9).
- **T9 (waitForLoadState)**: agregar el wait, re-correr el e2e, validar que pasa (GREEN).
- **T10, T11 (compose)**: no requieren tests (es config).
- **T12-T14 (docs)**: no requieren tests.

**T15 (verify)**: lo más importante. El verify phase corre `docker compose up`, espera `healthy`, y corre Playwright con `E2E_BASE_URL=http://localhost:3015`. Si pasa, el change es válido. Si falla, el refactor tiene un bug.

## Resultado

- **Status**: ok
- **Executive summary**: el change integra el frontend Qwik al stack dockerizado con static SSG + nginx (~30MB), ajusta el compose para tener un perfil VPS via override file, activa CORS en el Go bin via env var, y vuelve el e2e dual-mode via `E2E_BASE_URL`. El refactor de dos rutas (`routeLoader$` → `useVisibleTask$`) es la única superficie de código de aplicación. La imagen final del frontend es ~30MB (vs ~150MB Node alpine), alineado con la decisión de ultra-light footprint.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/proposal.md` (este archivo), engram topic `sdd/cachicamas-frontend-dockerize/proposal`.
- **Next recommended**: `sdd-spec` (producir los 6 specs delta con Given/When/Then).
- **Risks**: ver tabla (10 riesgos, 3 de severidad High: R1, R3, R4).
- **Skill resolution**: `paths-injected` (decisiones + archivos inyectados por el parent; el proposal los honra).
