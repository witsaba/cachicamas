# Explore: cachicamas-frontend-dockerize

> **Cambio**: `cachicamas-frontend-dockerize`
> **Status**: explored
> **Created**: 2026-07-03
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-frontend-dockerize/explore`)
> **Phase input**: ninguno (explore es la primera fase de un change nuevo)
> **Phase output**: este archivo (referencia para `sdd-proposal`)

---

## Resumen ejecutivo

El frontend Qwik (en `frontend/`) hoy no está en `docker-compose.yaml`: el servicio `database_administrator` (Go) corre en Docker pero el frontend se levanta en la terminal del desarrollador con `pnpm dev` (Vite dev server en `:5173`). Para un deploy en VPS, hace falta (a) empaquetar el frontend en una imagen Docker, (b) sumarlo al compose, (c) mantener la disciplina de pinning que ya tiene el repo, y (d) poder validar end-to-end con Playwright contra el stack dockerizado.

Decisiones del usuario (fijadas en preflight, no se re-debaten acá):

- **Modo de despliegue**: Static SSG + nginx. Imagen final ~30MB (vs ~150MB Node alpine). Sin runtime Node en producción.
- **Puerto externo**: 3015 (host) → 80 (container nginx).
- **Exposición en VPS**: solo el frontend publica puertos. Postgres, `database_administrator`, Jaeger y OTel collector quedan en la red privada `cachicamas_network`.
- **e2e target**: dual mode por `E2E_BASE_URL`. Sin la env var, Playwright sigue arrancando `pnpm dev` en 5173 (como hoy). Con la env var, no levanta `webServer` y apunta a la URL provista.

El hallazgo **más importante** de esta exploración es el **refactor obligatorio de `routeLoader$` a fetch client-side** en dos rutas: `/organizations` y `/organizations/{id}`. La razón es que Static SSG prerenderiza la HTML en build-time; el `routeLoader$` corre UNA vez en build y congela los datos. El test e2e crea una org y navega a su readback, así que la ruta dinámica tiene que poder resolver la org recién creada en runtime. La acción del form (`/organizations/new`) **no** necesita refactor: el handler ya corre en el browser vía un `$()` QRL que envuelve `createOrganization()` (fetch directo al Go bin con CORS).

El segundo hallazgo es que la **CORS del Go bin está en "producción = off"** (`backend/database_administrator/src/interfaces/http/cors.go`). El comentario en el código dice "the reverse proxy in front of the binary is expected to terminate on the same origin as the frontend". Como en el deploy del usuario **no hay reverse proxy** (nginx del frontend está en un contenedor separado), el browser del usuario va a pegarle al Go bin cross-origin. Hay que activar CORS vía `CORS_ALLOW_ORIGINS` con el origin del frontend (o el dominio del VPS).

---

## A. Topología actual del compose y qué cambia

Inventario de servicios en `docker-compose.yaml` (estado al 2026-07-03):

| Servicio | Imagen | Puertos publicados al host | Estado propuesto para VPS |
| ---------- | -------- | ------------------------------ | ----------------------------- |
| `postgres` | `postgres:18-alpine3.24` | `${POSTGRES_PORT:-5432}:5432` | **Quitar** el `ports:` (queda accesible solo dentro de `cachicamas_network`) |
| `jaeger` | `jaegertracing/jaeger:2.19.0` | `16686:16686`, `4317:4317`, `4318:4318` | **Quitar** todos los `ports:` |
| `otel-collector` | `otel/opentelemetry-collector-contrib:0.137.0` | `14317:4317`, `14318:4318`, `13133:13133` | **Quitar** todos los `ports:` |
| `database_administrator` | build local | `${SERVICE_PORT:-8080}:8080` | **Quitar** el `ports:` |
| `frontend` (NUEVO) | build local (`frontend/Dockerfile`) | — | `${FRONTEND_PORT:-3015}:80` (único público) |

Notas:

- El comentario en `postgres:5432` dice "publish for local debug". En VPS no aplica.
- El comentario en `jaeger:16686` dice "UI" — para un VPS de producción, exponerlo sería un riesgo de información. Queda privado.
- `database_administrator` en `:8080` solo lo necesita el browser que carga el frontend. Como el frontend en el compose va a estar en la misma red, puede llamarlo por DNS interno (`http://database_administrator:8080`).
- El "puerto host" `3015:80` es el **único** que queda publicado. El `80` interno es el default de nginx alpine.

**Decisión de scope a confirmar en `sdd-proposal`**: el compose único actual cumple los dos roles (local dev + VPS). Hay dos caminos:

1. **Compose único, parametrizado por env** — usar `${VPS_MODE:-false}` para condicional los `ports:`. Más simple, una sola fuente de verdad.
2. **Compose separado** — `docker-compose.yaml` (local dev, todo expuesto) + `docker-compose.vps.yaml` (override, solo frontend expuesto). Más limpio para audit, pero duplica mantenimiento.

El **recomendado** es el #1 (compose único parametrizado) porque el repo hoy tiene UN solo `docker-compose.yaml` y todos los cambios están documentados en comments inline. Mantener un solo archivo respeta la disciplina existente.

---

## B. Build de Qwik y feasibility del static SSG adapter

### B.1 Estado del adapter en `frontend/package.json`

```jsonc
"devDependencies": {
  "@builder.io/qwik": "^1.20.0",
  "@builder.io/qwik-city": "^1.20.0",
  // ... sin adapter instalado
}
```

**No hay** static adapter instalado. `pnpm qwik add static` lo agregaría como `@builder.io/qwik-city/adapters/static/vite`, modificaría `vite.config.ts` para registrar el adapter, y crearía `adapters/static/vite.config.ts`. La documentación oficial de Qwik City describe el adapter y los artefactos que produce.

Riesgo medio: `pnpm qwik add` puede ser interactivo (pregunta qué adapter). Hay que invocarlo con `--no-install` o similar, o preparar el cambio del `vite.config.ts` a mano. **A evaluar en la fase apply**.

### B.2 Artefactos que produce `qwik build` con el static adapter

Después de `pnpm qwik build` con el static adapter, se espera la siguiente estructura en `dist/`:

```
dist/
├── build/                  # client chunks (lazy-loaded)
│   └── q-*.js
├── assets/                 # assets estáticos
├── q-manifest.json         # manifest de chunks para resumability
├── qwik-plugins-manifest.json
└── ...                     # páginas prerenderizadas como .html
```

Y adicionalmente, según el adapter, se puede generar un `vercel.json`, `netlify.toml`, `static/_redirects`, etc. Para el static adapter puro, se generan archivos `.html` por cada ruta estática (`/`, `/organizations/new`).

### B.3 Rutas que se prerenderizan

El static adapter prerenderiza por default las rutas **estáticas**. Las rutas con segmentos dinámicos (`[id]`) requieren un hook `onStaticGenerate` que le diga al adapter qué valores concretos de `[id]` prerenderizar. **No es viable** prerenderizar IDs dinámicos para un dominio donde las orgs se crean en runtime.

Solución: nginx con **SPA fallback** (configurada más adelante en este explore) — todas las rutas no encontradas se sirven como `index.html`, y el cliente hace routing. La data dinámica se carga client-side (ver sección C).

---

## C. Audit de `routeLoader$` y `routeAction$` en `frontend/src/routes/`

Inspección de cada ruta. La conclusión por ruta:

### C.1 `/` — `frontend/src/routes/index.tsx`

- **Loader/action**: ninguno. Componente puro, sin fetch.
- **Refactor necesario**: **ninguno**.
- **SSG viable**: ✅ la página se prerenderiza como HTML estático sin data.

### C.2 `/organizations` — `frontend/src/routes/organizations/index.tsx`

- **Loader**: `useOrganizationsLoader = routeLoader$(...)` que llama `listOrganizations()` y devuelve `{ orgs, error? }`.
- **Refactor necesario**: **sí**. Convertir a client-side fetch con `useVisibleTask$` + `useSignal`. La data ya no viene del loader, sino de un signal que se llena después de hidratación.
- **Test impact**: el spec actual (`routes/organizations/index.spec.tsx`) renderiza el componente `OrganizationList` directamente con datos stub. **El test no toca el loader**, así que no se rompe. Hay que agregar (o ajustar) un test que verifique que la lista se carga via fetch cuando el loader ya no existe.
- **SSG viable**: ✅ con refactor.

### C.3 `/organizations/{id}` — `frontend/src/routes/organizations/[id]/index.tsx`

- **Loader**: `useOrganizationLoader = routeLoader$(async (event) => { ... })` que lee `event.params.id` y llama `getOrganization(id)`.
- **Refactor necesario**: **sí**. Mismo patrón: `useSignal` + `useVisibleTask$` que lee el `id` del URL (con `useLocation()`) y fetcha.
- **Test impact**: el spec actual (`routes/organizations/[id]/index.spec.tsx`) renderiza el componente `OrganizationReadback` directamente. **El test no toca el loader**, así que no se rompe. Hay que ajustar/ agregar un test que verifique el fetch client-side.
- **SSG viable**: ✅ con refactor (imposible prerenderizar IDs dinámicos sin `onStaticGenerate` restrictivo).

### C.4 `/organizations/new` — `frontend/src/routes/organizations/new/index.tsx`

- **Loader**: ninguno.
- **Action handler**: `submitAction = $(async (data) => { ... })` — un QRL **que ya corre en el browser**. El cuerpo llama `createOrganization()` de `~/lib/api` (un `fetch()` client-side con CORS).
- **Refactor necesario**: **ninguno**. La función ya es client-side; el QRL se shippea al browser y el browser hace el fetch al Go bin.
- **SSG viable**: ✅ la página se prerenderiza; la lógica del form corre en el browser.

### C.5 Conclusión del audit

| Ruta | Loader | Action | Refactor | Razón |
| ------ | -------- | -------- | ---------- | ------- |
| `/` | — | — | no | página estática pura |
| `/organizations` | `routeLoader$` | — | **sí** | data dinámica necesita client-side fetch |
| `/organizations/{id}` | `routeLoader$` | — | **sí** | id dinámico + data dinámica |
| `/organizations/new` | — | `$()` QRL (browser) | no | ya es client-side |

**Refactor scope estimado**: 2 archivos de rutas (`/organizations/index.tsx`, `/organizations/[id]/index.tsx`) + posiblemente ajustes en 2 specs (que ya renderizan el componente presentacional, así que probablemente **no**). Tests siguen verdes sin cambios. La función helper en `~/lib/api.ts` ya existe y ya devuelve `ApiResult<T>`.

**Patrón de refactor**:

```ts
// Antes (routeLoader$)
export const useOrganizationsLoader = routeLoader$(async () => {
  const result = await listOrganizations();
  if (result.ok) return { orgs: result.value.map(...) };
  return { orgs: [], error: result.message };
});

// Después (client-side)
import { useSignal, useVisibleTask$, $ } from "@builder.io/qwik";
export default component$(() => {
  const orgs = useSignal<OrganizationSummary[]>([]);
  const error = useSignal<string | null>(null);
  useVisibleTask$(async () => {
    const result = await listOrganizations();
    if (result.ok) {
      orgs.value = result.value.map(o => ({...}));
    } else {
      error.value = result.message;
    }
  });
  return <>...</>;
});
```

`useVisibleTask$` corre **solo en el client**, después de que el componente se vuelve visible. Es el patrón Qwik idiomático para "data que llega después del primer paint".

---

## D. Contrato CORS del Go bin

Archivo: `backend/database_administrator/src/interfaces/http/cors.go`.

Comportamiento del middleware:

- **Default en producción**: **CORS deshabilitado** (comentario explícito: "Production keeps this disabled — the reverse proxy in front of the binary is expected to terminate on the same origin as the frontend").
- **Default en dev (`SERVICE_ENV=development`)**: habilitado con `http://localhost:5173` como único origin.
- **Override**: `CORS_ALLOW_ORIGINS` (lista separada por comas) habilita CORS con esa lista, **independiente** de `SERVICE_ENV`.
- **No** se setea `Access-Control-Allow-Credentials` (decisión consciente).
- Preflight `OPTIONS` se responde con `204 No Content` sin invocar handlers downstream.

### Implicación para este change

Como el deploy del usuario **no usa reverse proxy** (el frontend estático en nginx y el Go bin son contenedores separados en la misma red compose), el browser va a hacer requests cross-origin. **Hay que activar CORS en el Go bin** vía `CORS_ALLOW_ORIGINS`.

**Configuración propuesta** (a confirmar en `sdd-design`):

```yaml
# docker-compose.yaml — servicio database_administrator
environment:
  CORS_ALLOW_ORIGINS: ${CORS_ALLOW_ORIGINS:-http://localhost:3015}
```

Y `.env.example` debería tener:

```bash
# Origin del frontend (CORS allowlist del Go bin).
# Para deploy en VPS, cambiar al dominio público: https://cachicamas.example.com
CORS_ALLOW_ORIGINS=http://localhost:3015
```

El frontend envía el header `Origin: <origen>` automáticamente. El browser bloquea si la respuesta no trae `Access-Control-Allow-Origin: <mismo origin>`. El middleware ya está implementado y testado (regresión de CORS del 2026-07-03 ya está cubierta por el e2e existente).

### Caso especial: el `PUBLIC_API_BASE_URL` que el frontend usa

`src/lib/api.ts:apiBaseUrl()` lee `process.env.PUBLIC_API_BASE_URL` con default `http://localhost:8080`. En el compose, **dentro de la red** el frontend puede llamarlo por DNS interno: `http://database_administrator:8080`. En el browser (no en el container), `PUBLIC_API_BASE_URL` se inlinea en build-time (Vite `PUBLIC_*`), así que el `build` del frontend debe setearlo.

```yaml
# docker-compose.yaml — servicio frontend (build args)
build:
  context: ./frontend
  args:
    PUBLIC_API_BASE_URL: ${PUBLIC_API_BASE_URL:-http://localhost:8080}
```

**Tradeoff**: con `PUBLIC_API_BASE_URL=http://localhost:8080` (default), el browser del usuario en el VPS asume que el Go bin está en `localhost:8080` del host. Eso no es cierto en VPS (Go bin está en la red privada). Soluciones:

1. **Reverse proxy**: nginx en el frontend también proxy-ea `/api/*` al Go bin. Una sola origin. (Complica nginx, agrega hop.)
2. **DNS público**: que el Go bin quede expuesto también (descartado,违背 "VPS exposure = solo frontend").
3. **`PUBLIC_API_BASE_URL` por entorno**: build del frontend con el origin correcto del VPS. Para dev local: `http://localhost:8080`. Para VPS: el dominio público del Go bin (si lo hubiera) o un path proxy-eado por nginx.

**Recomendación**: opción #3 con nginx como reverse proxy **solo para `/api/*`** (mantiene simple el routing, una sola origin para el browser, CORS innecesario en el Go bin). Esto se discute en `sdd-design`.

---

## E. Setup del e2e (Playwright)

Estado actual de `frontend/playwright.config.ts`:

```ts
export default defineConfig({
  testDir: "./e2e",
  use: { baseURL: "http://localhost:5173" },
  webServer: {
    command: "pnpm dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
  // ...
});
```

### Cambio necesario

Convertir `baseURL` y `webServer` a env-driven:

```ts
const e2eBaseUrl = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const useDevServer = !process.env.E2E_BASE_URL;

export default defineConfig({
  testDir: "./e2e",
  use: { baseURL: e2eBaseUrl, /* ... */ },
  webServer: useDevServer
    ? {
        command: "pnpm dev",
        url: "http://localhost:5173",
        reuseExistingServer: !process.env.CI,
        timeout: 60_000,
      }
    : undefined,
  // ...
});
```

### Comportamiento esperado

- **Dev local sin Docker**: `pnpm test:e2e` (sin env var) → `baseURL: http://localhost:5173`, levanta `pnpm dev` (igual que hoy).
- **CI / post-deploy en compose**: `E2E_BASE_URL=http://localhost:3015 pnpm test:e2e` → `baseURL: http://localhost:3015`, **no** levanta `webServer` (confía en que el compose ya corre el Qwik). El spec sigue siendo el mismo (`e2e/create-organization.spec.ts`); solo cambia dónde apunta.
- **VPS producción (futuro)**: `E2E_BASE_URL=https://cachicamas.example.com pnpm test:e2e` → apunta al dominio público, valida TLS, headers reales, etc.

### Compatibilidad con el spec existente

`frontend/e2e/create-organization.spec.ts` (líneas 1-100 ya leídas):

- `test("submit form persists to Postgres and navigates to detail page", ...)` — este spec crea una org vía el form y verifica la navegación a `/organizations/{id}`.
- Asume que `routeAction$` corre en el browser con CORS al Go bin.
- Asume que la página de readback (`/organizations/{id}`) muestra la org recién creada.

**Con el refactor de `routeLoader$` a client-side fetch** (sección C), el readback hace un fetch client-side al Go bin. El spec debería seguir pasando, pero **hay que verificar** en `sdd-verify` que la navegación a `/organizations/{id}` no rompe (puede haber un flash de "loading" o un estado vacío mientras llega el fetch — el spec actual no espera nada, simplemente navega y verifica contenido).

### Cambio opcional recomendado (no bloqueante)

Agregar `waitForLoadState("networkidle")` en la navegación al readback, para asegurar que el fetch client-side se completó antes de los asserts. Es una mejora de robustness del test, no un cambio funcional.

---

## F. `.env` y `.env.example`

El archivo `.env.example` está en la lista de denegados por la safety policy; no se pudo leer su contenido. La inspección se hace por inferencia desde `docker-compose.yaml` y los `README.md` que ya leí.

Variables que el compose actual consume (de los `${VAR:-default}` en el YAML):

- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`, `POSTGRES_INITDB_ARGS`, `QUEEN_PASSWORD` — para Postgres.
- `JAEGER_UI_PORT`, `JAEGER_OTLP_GRPC_PORT`, `JAEGER_OTLP_HTTP_PORT` — para Jaeger.
- `OTEL_COLLECTOR_GRPC_PORT`, `OTEL_COLLECTOR_HTTP_PORT`, `OTEL_COLLECTOR_HEALTH_PORT` — para OTel collector.
- `SERVICE_ENV`, `SERVICE_PORT`, `PROJECT_NAME` — para `database_administrator`.

Variables nuevas que este change introduce:

- `CORS_ALLOW_ORIGINS` — para el Go bin. Default: `http://localhost:3015`.
- `FRONTEND_PORT` — host port. Default: `3015`.
- `PUBLIC_API_BASE_URL` — para el build de Qwik (build arg, no env runtime). Default: depende de la decisión de D (reverse proxy o dominio público).

`.env.example` debe actualizarse para documentar las nuevas variables (este es un cambio esperado, no un stretch).

---

## G. Disciplina de pinning

El compose actual usa **pinned-by-tag** (no `latest`, no `alpine` flotante). Ejemplos:

- `postgres:18-alpine3.24`
- `jaegertracing/jaeger:2.19.0`
- `otel/opentelemetry-collector-contrib:0.137.0`

Nota: el `README.md` menciona "triple-pinned" en el contexto de `project.md`, pero el YAML real no usa digest (`@sha256:...`); usa **tag + flavor explícito** (e.g., `alpine3.24` en lugar de `alpine`). Eso es pinning de un solo nivel. Aceptable para el scope de este change; un pinning con digest es un follow-up aparte.

### Imagen propuesta para nginx

`nginx:1.27-alpine`. Pinning explícito (no `:latest`, no `:alpine` flotante). Versión actual estable de nginx es 1.27.x; verificar el patch exacto en el momento de apply. Como el resto del compose, **NO** usar digest en este change (consistencia con el resto del repo).

### Imagen para el builder

`node:20-alpine` (la versión de Node que ya requiere `engines` en `frontend/package.json`: `^18.17.0 || ^20.3.0 || >=21.0.0`). El pin exacto debería ser `node:20.x.y-alpine` o `node:20.x.y-alpine3.x`. Para empezar: `node:20-alpine` es aceptable; refinar a `node:20.19.0-alpine` si la disciplina del repo lo exige.

---

## H. Healthcheck strategy

`docker-compose.yaml` actual tiene dos patrones:

- `postgres`, `jaeger`, `otel-collector` — `healthcheck` con `CMD-SHELL` o `wget --spider` (porque sus imágenes tienen shell + wget).
- `database_administrator` — `healthcheck.disable: true` (porque la imagen es distroless/static, sin shell, sin wget, sin curl). El comentario en el YAML documenta el por qué.

### Healthcheck propuesto para `frontend` (nginx alpine)

`nginx:1.27-alpine` viene con `wget` (busybox). El healthcheck puede ser:

```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://127.0.0.1/"]
  interval: 10s
  timeout: 5s
  retries: 6
  start_period: 10s
```

(Patrón idéntico al de `jaeger` en el compose actual.)

Riesgo bajo: `wget` en busybox a veces se comporta raro con `localhost`; usar `127.0.0.1` explícito (mismo comentario que en el YAML de jaeger).

---

## I. Riesgos y sorpresas

| # | Riesgo | Severidad | Mitigación |
| --- | -------- | ----------- | ------------ |
| R1 | **Refactor de `routeLoader$` rompe la página de readback** durante la navegación post-create-organization del e2e | **High** | Tests existentes renderizan el componente directamente con stub, así que el refactor del loader no rompe los specs viejos. `sdd-verify` debe correr el e2e end-to-end contra el stack dockerizado y reportar el resultado. Si falla, agregar `waitForLoadState("networkidle")` antes de los asserts. |
| R2 | `pnpm qwik add static` es interactivo o modifica `vite.config.ts` de formas no esperadas | Medium | Revisar el diff que produce el comando. Si es no-determinístico, escribir el `adapters/static/vite.config.ts` a mano y registrar el adapter en `vite.config.ts` con un patch explícito. |
| R3 | El browser del usuario en el VPS hace requests cross-origin al Go bin y los bloquea por CORS | **High** | Configurar `CORS_ALLOW_ORIGINS` con el origin del frontend (o del dominio público). Documentar en `.env.example`. Verificar en `sdd-verify` con un test que dispare un fetch desde el browser. |
| R4 | `PUBLIC_API_BASE_URL` se inlinea en build-time; si el default es `localhost:8080`, el browser en el VPS no llega al Go bin | **High** | Decidir en `sdd-design` entre (a) nginx reverse-proxy `/api/*` al Go bin (más simple, CORS innecesario), (b) build por entorno con el origin correcto, (c) otra alternativa. |
| R5 | El Qwik static adapter no genera `index.html` para rutas con segmentos dinámicos → nginx devuelve 404 | Medium | Configurar nginx con SPA fallback: `try_files $uri $uri/ /index.html;`. El cliente maneja el routing. La data dinámica se carga client-side via el refactor de R1. |
| R6 | El e2e `create-organization.spec.ts` no espera a que la data se cargue en `/organizations/{id}` (porque antes venía del loader, ahora viene de un fetch client-side) | Medium | Agregar `waitForLoadState("networkidle")` o `expect(...).toBeVisible({ timeout: 5000 })` en el spec. |
| R7 | El `entry.preview.tsx` y los `entry.ssr.tsx` que existen en `frontend/src/` se confunden con el `entry.express.tsx` o similar que necesitaría el static adapter | Low | El static adapter no necesita un entry point de runtime (es estático puro). Los archivos existentes pueden quedarse o limpiarse en un follow-up; no bloquean este change. |
| R8 | La imagen nginx default no tiene gzip ni cache headers optimizados | Low | Configurar nginx.conf con `gzip on`, `gzip_types text/plain text/css application/json application/javascript`, y cache headers de 1 año para `/build/` (chunks Qwik inmutables). Mejora performance, no bloquea. |
| R9 | El container `frontend` no tiene `depends_on: database_administrator: condition: service_healthy` y arranca antes de que el Go bin esté listo | Low | El `healthcheck` del Go bin está `disable: true` (por distroless), así que `depends_on: condition: service_healthy` no funcionaría. Usar `depends_on: database_administrator: condition: service_started` o nada (la SPA falla al primer POST, pero reintenta al render del readback). Aceptable. |
| R10 | El Qwik static adapter requiere un `qwikCity({ trailingSlash: ... })` config para que nginx SPA fallback funcione bien | Low | Verificar el comportamiento por defecto. Si las URLs no tienen trailing slash, configurar nginx para redirigir o servir el mismo HTML. |

---

## Affected Areas (resumen para `sdd-proposal`)

| Area | Tipo de cambio | Descripción |
| ------ | ---------------- | ------------- |
| `frontend/Dockerfile` | **New** | Multi-stage: builder (node:20-alpine + pnpm) → runner (nginx:1.27-alpine). |
| `frontend/nginx.conf` | **New** | SPA fallback + gzip + cache headers. |
| `frontend/vite.config.ts` | **Modified** | Agregar `qwikCity({ ... })` con static adapter, importar `adapters/static/vite.config.ts`. |
| `frontend/src/routes/organizations/index.tsx` | **Modified** | Refactor: `routeLoader$` → `useVisibleTask$` + `useSignal`. |
| `frontend/src/routes/organizations/[id]/index.tsx` | **Modified** | Refactor: `routeLoader$` → `useVisibleTask$` + `useSignal` + `useLocation()`. |
| `frontend/src/routes/organizations/index.spec.tsx` | **Modified (opcional)** | Verificar que sigue testeando el componente (no el loader). |
| `frontend/src/routes/organizations/[id]/index.spec.tsx` | **Modified (opcional)** | Igual. |
| `frontend/e2e/create-organization.spec.ts` | **Modified (opcional)** | Agregar `waitForLoadState` después de la navegación al readback. |
| `frontend/playwright.config.ts` | **Modified** | `baseURL` y `webServer` env-driven (`E2E_BASE_URL`). |
| `docker-compose.yaml` | **Modified** | Agregar servicio `frontend`; parametrizar todos los `ports:` por `VPS_MODE` (o separar en override). |
| `backend/database_administrator/src/interfaces/http/cors.go` | **Modified** | El default en dev pasa a ser `http://localhost:5173,http://localhost:3015` cuando `CORS_ALLOW_ORIGINS` no está seteado. **OR** (preferido) dejar el default en dev intacto y forzar al usuario a setear `CORS_ALLOW_ORIGINS` en el `.env` del VPS. |
| `.env.example` | **Modified** | Documentar `CORS_ALLOW_ORIGINS`, `FRONTEND_PORT`, `PUBLIC_API_BASE_URL`. |
| `README.md` | **Modified** | Una nota de 3-5 líneas en la sección de deploy explicando el cambio de topology y la nueva variable `FRONTEND_PORT`. |
| `frontend/README.md` | **Modified** | Sección "End-to-end tests" con la nota del nuevo flujo `E2E_BASE_URL`. |

---

## Notas para la fase `sdd-proposal`

El proposal debe:

1. Fijar las decisiones tomadas en preflight (Static SSG + nginx, puerto 3015, VPS exposure = solo frontend, e2e dual mode).
2. Plantear el refactor de `routeLoader$` como una **decisión arquitectónica explícita** (no como tarea menor), con justificación y plan de rollback (revertir a `routeLoader$` es trivial: solo el archivo de la ruta).
3. Decidir entre las opciones de `PUBLIC_API_BASE_URL` (sección D): **(recomendado)** nginx reverse-proxy `/api/*` al Go bin — una sola origin para el browser, CORS innecesario en el Go bin, build del frontend siempre con `PUBLIC_API_BASE_URL=/api/v1` o similar.
4. Plantear el tradeoff del "compose único vs separado" para VPS (sección A): recomendado parametrizar por env.
5. Listar las capabilities nuevas y modificadas (siguiendo el formato del proposal de `cachicamas-tail-sampling`).
6. Plan de rollback: `git revert` del PR + `docker compose up -d --build`. Sin schema changes, sin migraciones.

## Notas para la fase `sdd-spec`

Specs a producir en `openspec/changes/cachicamas-frontend-dockerize/specs/`:

- `frontend-docker-image/spec.md` — el Dockerfile produce una imagen reproducible con nginx + dist/.
- `frontend-spa-fallback/spec.md` — nginx sirve todas las rutas no encontradas como `index.html`; el cliente hace routing.
- `frontend-cors-contract/spec.md` — el browser del frontend puede llamar al Go bin cross-origin sin error de CORS.
- `frontend-e2e-dual-mode/spec.md` — `pnpm test:e2e` corre contra el dev server; `E2E_BASE_URL=...` corre contra el compose.
- `organizations-client-side-data/spec.md` — las rutas `/organizations` y `/organizations/{id}` cargan la data dinámicamente desde el browser; sin `routeLoader$`.

## Notas para la fase `sdd-design`

Puntos a diseñar:

- **Topología final del compose** (con o sin reverse proxy interno, VPS exposure).
- **Multi-stage Dockerfile** (node version exacta, pnpm install flags, qué se copia, ownership de archivos, permisos).
- **nginx.conf completo** (SPA fallback, gzip, cache headers, healthcheck endpoint opcional).
- **vite.config.ts** patch (cómo registrar el static adapter sin `pnpm qwik add` interactivo).
- **Refactor de las dos rutas** (diff exacto, helper extraído si vale la pena).
- **CORS config** (`CORS_ALLOW_ORIGINS` default, comportamiento del Go bin en prod con `SERVICE_ENV=production`).
- **.env.example diff** (antes/después de cada variable).

## Notas para la fase `sdd-tasks`

Forecast del budget 400-line: **Medium** (estimado: 200-350 líneas de diff, repartido entre el Dockerfile, nginx.conf, vite.config.ts, dos rutas, playwright.config.ts, docker-compose.yaml, .env.example, README). Single PR recomendado. **Chained PR** solo si el refactor de las dos rutas resulta en > 200 líneas de cambios en `frontend/src/routes/` (improbable según lo estimado).

Tareas tentativas (refinar en `sdd-tasks`):

- T1: `frontend/Dockerfile` multi-stage con `node:20-alpine` + `nginx:1.27-alpine`.
- T2: `frontend/nginx.conf` con SPA fallback, gzip, cache headers.
- T3: Instalar/configurar el static adapter de Qwik en `vite.config.ts`.
- T4: Refactor `/organizations` loader → client-side fetch.
- T5: Refactor `/organizations/{id}` loader → client-side fetch.
- T6: Refactor `playwright.config.ts` a env-driven.
- T7: Refactor `docker-compose.yaml` para incluir el servicio `frontend` + parametrizar `ports:` por `VPS_MODE`.
- T8: Actualizar `.env.example` con `CORS_ALLOW_ORIGINS`, `FRONTEND_PORT`, `PUBLIC_API_BASE_URL`.
- T9: Documentar en `README.md` (raíz) y `frontend/README.md` el nuevo flujo de e2e y la variable `FRONTEND_PORT`.
- T10: `sdd-verify`: `docker compose up -d --build`, `curl -fsS http://localhost:3015/`, `pnpm test:e2e` con `E2E_BASE_URL=http://localhost:3015`, validar CORS con un test de browser que dispare un fetch.

## Notas para la fase `sdd-apply`

Strict TDD (per `openspec/config.yaml`): aplica al Go service. Para el frontend, el patrón de Vitest + Playwright ya está vigente. La regla práctica:

- Tests unitarios nuevos o modificados en `frontend/src/**/*.spec.{ts,tsx}` deben escribirse **antes** del código que los hace pasar.
- El e2e spec se modifica antes del cambio de `playwright.config.ts` y se verifica que falla (RED), luego se hace pasar (GREEN).
- El refactor de las dos rutas no necesita nuevos specs si los existentes siguen testeando el componente presentacional (que es el caso según lo leído).

## Resultado

- **Status**: ok
- **Executive summary**: la decisión de Static SSG + nginx es viable y ultra-light (~30MB), pero requiere un refactor acotado de `routeLoader$` a client-side fetch en dos rutas. La acción del form ya es client-side. CORS necesita activarse explícitamente vía `CORS_ALLOW_ORIGINS` porque el deploy en VPS no tiene reverse proxy que termine en la misma origin. El e2e se vuelve dual-mode con un cambio mínimo en `playwright.config.ts`.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/explore.md` (este archivo), engram topic `sdd/cachicamas-frontend-dockerize/explore`.
- **Next recommended**: `sdd-proposal` (siempre después de un explore exitoso).
- **Risks**: ver sección I (10 riesgos enumerados, 3 de severidad High).
- **Skill resolution**: `paths-injected` (el parent pasó las decisiones + archivos; este explore los usó como restricciones).
