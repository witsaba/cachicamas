# Design: cachicamas-frontend-dockerize

> **Cambio**: `cachicamas-frontend-dockerize`
> **Status**: designed
> **Created**: 2026-07-03
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-frontend-dockerize/design`)
> **Phase inputs**: `proposal.md` + 3 specs en `specs/*/spec.md`
> **Phase output**: este archivo (referencia para `sdd-tasks`)

---

## 1. Visión general de la solución

El change introduce una imagen Docker para el frontend Qwik (Static SSG servido por nginx) y un override file de docker-compose que produce un perfil "VPS" donde solo el frontend expone un puerto al host. Internamente, nginx hace reverse-proxy de `/api/*` al Go bin para que el browser solo vea un origin. CORS queda configurable vía `CORS_ALLOW_ORIGINS` para casos de borde (debugging, gateways futuros). El refactor de `routeLoader$` a `useVisibleTask$` mantiene la funcionalidad visible idéntica. El e2e runner se vuelve dual-mode con un patch mínimo en `playwright.config.ts`.

**Decisiones de diseño clave** (heredadas de la proposal; este documento las aterriza):

- D1: Multi-stage Dockerfile con `node:20-alpine` (builder) + `nginx:1.27-alpine` (runner). Imagen final ~30MB.
- D2: Static adapter de Qwik escrito a mano (no `pnpm qwik add` interactivo) en `frontend/adapters/static/vite.config.ts`, registrado en `frontend/vite.config.ts`.
- D3: nginx como reverse proxy interno de `/api/*` → `http://database_administrator:8080/` (sin el prefijo `/api`).
- D4: Refactor de `routeLoader$` a `useVisibleTask$` + `useSignal` en dos rutas. Cero cambios en los componentes presentacionales ni en los specs Vitest existentes.
- D5: docker-compose.vps.yaml como override file (no parametrización por `VPS_MODE` env var). Justificación: YAML no permite condicionales de bloque; profiles de compose no ocultan `ports:`.
- D6: CORS del Go bin sin cambios de código. Override file setea `SERVICE_ENV=production` + `CORS_ALLOW_ORIGINS` env var.
- D7: playwright.config.ts con `E2E_BASE_URL` env var; webServer condicional.

---

## 2. Diagrama de secuencia end-to-end

Flujo "create organization" desde el browser del usuario, pasando por todos los componentes del stack dockerizado:

```text
┌────────┐   1. POST /api/organizations (form-encoded)
│Browser │──────────────────────────────────────────────────┐
│  :5173 │   Origin: http://localhost:3015                  │
│ (test) │   Body: full_name=Acme&identification=acme        │
└────────┘                                                     ▼
                                                         ┌──────────────┐
                                                         │ nginx        │
                                                         │ :80          │
                                                         │ (frontend)   │
                                                         │              │
                                                         │ location /api│
                                                         │   proxy_pass │
                                                         └──────┬───────┘
                                                                │ 2. POST /organizations
                                                                │    (no /api prefix)
                                                                │    Host: database_administrator
                                                                ▼
                                                         ┌──────────────────────┐
                                                         │ database_administrator│
                                                         │ :8080                 │
                                                         │ (Go, internal)        │
                                                         │                       │
                                                         │ CORS:                 │
                                                         │  - Origin:            │
                                                         │    http://localhost:3 │
                                                         │    015 (in allowlist)  │
                                                         │  - Returns:            │
                                                         │    ACAO: http://...    │
                                                         └──────┬────────────────┘
                                                                │ 3. INSERT INTO organization
                                                                │    (queen role)
                                                                ▼
                                                         ┌──────────────┐
                                                         │ postgres     │
                                                         │ :5432        │
                                                         │ (internal)   │
                                                         └──────┬───────┘
                                                                │ 4. RETURNING id, ...
                                                                ▼
                                                         (back through Go → nginx)
                                                                │
                                                                ▼
┌────────┐   5. 201 Created { id: 1, ... }                ┌──────────────┐
│Browser │◄─────────────────────────────────────────────│ nginx         │
│        │   + Access-Control-Allow-Origin: ...          │               │
└────────┘                                               └───────────────┘
   │
   │ 6. window.location = '/organizations/1'
   │
   ▼
┌────────┐   7. GET /organizations/1 (HTML)                ┌──────────────┐
│Browser │─────────────────────────────────────────────►│ nginx         │
│        │                                                │ SPA fallback  │
│        │◄─────────────────────────────────────────────│ → index.html  │
└────────┘   8. HTML shell (Qwik boot)                    └──────────────┘
   │
   │ 9. Qwik hydrates, useVisibleTask$ fires
   │
   ▼
┌────────┐   10. GET /api/organizations/1                   ┌──────────────┐
│Browser │──────────────────────────────────────────────►│ nginx         │
│        │                                                │ /api proxy    │
│        │◄──────────────────────────────────────────────│               │
└────────┘   11. { id: 1, full_name: "Acme", ... }          └──────────────┘
   │
   │ 12. Render <OrganizationReadback org={...} />
   ▼
(readback visible)
```

**Notas sobre el flujo:**

- En el paso 1, el browser está testeando desde el host (`localhost:3015` mapeado a `3015` host → `80` container). En VPS, sería `https://cachicamas.example.com`.
- En el paso 2, nginx **strippea** el prefijo `/api` (ver §4).
- En el paso 5, la respuesta **no requiere** CORS headers en el flujo normal (mismo origin), pero nginx los pasa transparentes si están. El Go los setea igual porque la request **sí** trae `Origin: http://localhost:3015`.
- En el paso 7, el browser pide `/organizations/1` y nginx **no encuentra** un archivo estático para esa ruta (es dinámica). SPA fallback sirve `index.html` (paso 8).
- En el paso 10, la segunda navegación (post-`window.location = ...`) sí requiere CORS si el browser considera cross-origin. Como nginx es el mismo origin, **no hay** cross-origin → CORS irrelevante en este path. El Go igual responde con los headers.

---

## 3. Multi-stage Dockerfile

### 3.1 Diff completo de `frontend/Dockerfile` (NUEVO)

```dockerfile
# syntax=docker/dockerfile:1.7
#
# Cachicamas frontend — multi-stage build.
#
# Stage 1: builder
#   - node:20-alpine
#   - pnpm install --frozen-lockfile (deterministic)
#   - pnpm build → dist/ (Qwik static SSG output)
#
# Stage 2: runner
#   - nginx:1.27-alpine
#   - copia dist/ a /usr/share/nginx/html
#   - copia nginx.conf custom
#   - healthcheck via wget
#
# Output: ~30MB image. Sin Node, sin pnpm, sin source en el runner.

# ─── Stage 1: builder ────────────────────────────────────────────────────
FROM node:20-alpine AS builder

# pnpm via corepack (recomendado por la doc oficial de Node 20+).
# corepack ships con node:20-alpine; no requiere npm install -g.
RUN corepack enable

WORKDIR /app

# Copiamos solo los manifests primero para aprovechar el cache de Docker
# (pnpm install cambia rara vez; src/ cambia frecuente).
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# Recién ahora copiamos el resto (cambios en src/ no invalidan el install).
COPY . .

# Build arg para que el browser sepa adonde pegarle. Default: /api
# (nginx reverse-proxy). Override en docker-compose con PUBLIC_API_BASE_URL.
ARG PUBLIC_API_BASE_URL=/api
ENV PUBLIC_API_BASE_URL=${PUBLIC_API_BASE_URL}

# qwik build produce dist/ con los assets estáticos + las rutas estáticas
# prerenderizadas. Las rutas dinámicas (/organizations/{id}) NO se incluyen
# — quedan para SPA fallback (ver §4).
RUN pnpm build

# ─── Stage 2: runner ────────────────────────────────────────────────────
FROM nginx:1.27-alpine

# Copiamos los assets estáticos al document root de nginx.
COPY --from=builder /app/dist /usr/share/nginx/html

# Config custom de nginx (SPA fallback + reverse proxy /api).
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Exponemos el puerto 80 (interno; el mapping al host se hace en compose).
EXPOSE 80

# Healthcheck via wget (busybox). Mismo patrón que el servicio jaeger
# del compose actual.
HEALTHCHECK --interval=10s --timeout=5s --retries=6 --start-period=10s \
  CMD wget --spider -q http://127.0.0.1/ || exit 1

# nginx en foreground (default del entrypoint de la imagen; explícito para
# evitar drift si en el futuro alguien cambia la base).
CMD ["nginx", "-g", "daemon off;"]
```

### 3.2 Diff completo de `frontend/.dockerignore` (NUEVO)

```gitignore
node_modules
dist
test-results
tmp
e2e/test-results
.atl
.git
.gitignore
.DS_Store
*.log
.env.local
.env.*.local
playwright-report
```

### 3.3 Rationale del Dockerfile

| Decisión | Alternativa descartada | Por qué |
| ---------- | ------------------------ | --------- |
| `node:20-alpine` builder | `node:20-slim` (debian) | alpine es ~50MB vs ~250MB; suficiente para pnpm |
| `nginx:1.27-alpine` runner | `nginxinc/nginx-unprivileged` | el actual compose ya usa `nginx:1.27-alpine`; consistencia |
| `corepack enable` para pnpm | `npm install -g pnpm` | corepack ships con Node 20+; respeta `packageManager` del package.json |
| `pnpm install --frozen-lockfile` | `pnpm install` | garantiza reproducibilidad (CI-friendly) |
| Build arg `PUBLIC_API_BASE_URL` | hardcoded en nginx.conf | vite lo inlinea en build-time; no se puede cambiar post-build |
| COPY manifests antes que src/ | COPY todo junto | maximiza cache hit (install layer no se invalida con cambios de src) |
| `wget --spider` healthcheck | sin healthcheck | consistente con jaeger; permite `depends_on: service_healthy` (no usado aquí porque el Go bin no tiene healthcheck, pero disponible) |

---

## 4. nginx.conf

### 4.1 Diff completo de `frontend/nginx.conf` (NUEVO)

```nginx
# Cachicamas frontend — nginx config.
#
# - SPA fallback: rutas no encontradas → /index.html (para client-side routing).
# - Reverse proxy /api/* → database_administrator:8080 (sin /api prefix).
# - Gzip para text assets.
# - Cache 1y para /build/* (chunks Qwik inmutables, content-hashed).
# - Cache no-store para /index.html (el shell siempre se sirve fresh).

server {
  listen 80;
  server_name _;

  # Logs al stdout/stderr (docker logs los captura). json格式 sería ideal
  # para Loki, pero suma complejidad fuera de scope.
  access_log /var/log/nginx/access.log;
  error_log /var/log/nginx/error.log;

  # Gzip para text-based responses.
  gzip on;
  gzip_vary on;
  gzip_min_length 1024;
  gzip_proxied any;
  gzip_types
    text/plain
    text/css
    text/xml
    text/javascript
    application/json
    application/javascript
    application/xml
    application/xml+rss
    image/svg+xml;

  # ─── SPA shell ──────────────────────────────────────────────────────
  # El index.html NO se cachea (queremos que las deploys se vean fresh).
  location = /index.html {
    add_header Cache-Control "no-cache, no-store, must-revalidate" always;
    add_header Pragma "no-cache" always;
    add_header Expires "0" always;
  }

  # ─── Static assets con hash ─────────────────────────────────────────
  # Qwik genera filenames con content-hash (e.g., q-abc123.js). Cache
  # agresivo: cuando cambia el contenido, cambia el filename.
  location /build/ {
    expires 1y;
    add_header Cache-Control "public, max-age=31536000, immutable" always;
    try_files $uri =404;
  }

  # Otros assets estáticos (favicon, fonts, etc.) — cache moderado.
  location ~* \.(ico|svg|png|jpg|jpeg|gif|webp|woff2?)$ {
    expires 30d;
    add_header Cache-Control "public, max-age=2592000" always;
    try_files $uri =404;
  }

  # ─── API reverse proxy ──────────────────────────────────────────────
  # /api/* → http://database_administrator:8080/ (sin el /api prefix).
  # Ejemplo: /api/organizations/1 → /organizations/1 en el Go bin.
  #
  # Strip del prefix: proxy_pass con trailing slash (`/api/`) hace que
  # nginx reemplace el matching part del location con la del proxy_pass.
  # Más detalles: https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass
  location /api/ {
    proxy_pass http://database_administrator:8080/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header Connection "";

    # Timeouts razonables para un dev/VPS típico.
    proxy_connect_timeout 5s;
    proxy_send_timeout 30s;
    proxy_read_timeout 30s;

    # No buffering de respuestas del Go bin — streaming directo al browser.
    proxy_buffering off;
  }

  # ─── SPA fallback ───────────────────────────────────────────────────
  # Cualquier ruta no matcheada por las locations de arriba → /index.html.
  # El cliente hace routing (Qwik City client-side router).
  location / {
    try_files $uri $uri/ /index.html;
  }
}
```

### 4.2 Por qué `/api/` con trailing slash

`location /api/` con `proxy_pass http://database_administrator:8080/` (también con trailing slash) hace que nginx **reemplace** el prefijo `/api/` con `/` antes de hacer el proxy_pass. Resultado: el Go bin ve la URL sin el prefijo.

| Browser pide | nginx matchea | proxy_pass genera | Go bin ve |
| -------------- | --------------- | ------------------- | ----------- |
| `/api/organizations` | `location /api/` | `http://database_administrator:8080/organizations` | `/organizations` |
| `/api/organizations/1` | `location /api/` | `http://database_administrator:8080/organizations/1` | `/organizations/1` |
| `/api/health` | `location /api/` | `http://database_administrator:8080/health` | `/health` |

Esto es el patrón canónico de nginx para "prefix-stripping reverse proxy". La alternativa (sin trailing slash en `proxy_pass`) **conservaría** el `/api` y el Go tendría que montar el router con ese prefijo. Strippear acá es más limpio.

### 4.3 Por qué `try_files $uri $uri/ /index.html;`

Tres intentos en orden:

1. `$uri` — archivo estático en `/usr/share/nginx/html/` (e.g., `dist/organizations/index.html`).
2. `$uri/` — directorio (e.g., `dist/organizations/` → busca `index.html` adentro).
3. `/index.html` — fallback SPA.

Si Qwik prerenderiza `/organizations` como `dist/organizations/index.html`, el primer intento pega y devuelve el HTML correcto. Si pide `/organizations/123` (ruta dinámica), el primer y segundo intento fallan, el tercero devuelve `index.html` y el cliente hace routing.

---

## 5. Static adapter de Qwik

### 5.1 Diff de `frontend/adapters/static/vite.config.ts` (NUEVO)

```typescript
/**
 * Static adapter config para Qwik City.
 *
 * Genera archivos HTML estáticos para las rutas prerenderizables
 * y deja las dinámicas para SPA fallback. El output va a dist/.
 *
 * Fuente: https://qwik.dev/qwikcity/guides/static-site-generation/
 *
 * El adapter viene como sub-export de @builder.io/qwik-city
 * (que ya está en devDependencies), así que no requiere
 * agregar una dep nueva.
 */
import { staticAdapter } from "@builder.io/qwik-city/adapters/static/vite";
import { extendConfig } from "@builder.io/qwik-city/vite";
import baseConfig from "../../vite.config";

export default extendConfig(baseConfig, () => {
  return {
    build: {
      ssr: true,
      rollupOptions: {
        input: ["@qwik-city-plan"],
      },
    },
    plugins: [
      staticAdapter({
        origin: "http://localhost:3015",
      }),
    ],
  };
});
```

### 5.2 Diff de `frontend/vite.config.ts` (MODIFIED)

Patch mínimo: importar el adapter config cuando `command === "build"`. El archivo base NO se toca para `pnpm dev`.

```typescript
// … imports existentes …
import { defineConfig, type UserConfig } from "vitest/config";
import { qwikVite } from "@builder.io/qwik/optimizer";
import { qwikCity } from "@builder.io/qwik-city/vite";
import tsconfigPaths from "vite-tsconfig-paths";
import pkg from "./package.json";
import tailwindcss from "@tailwindcss/vite";

type PkgDep = Record<string, string>;
const { dependencies = {}, devDependencies = {} } = pkg as any as {
  dependencies: PkgDep;
  devDependencies: PkgDep;
  [key: string]: unknown;
};
errorOnDuplicatesPkgDeps(devDependencies, dependencies);

export default defineConfig(({ command, mode }): UserConfig => {
  return {
    plugins: [
      qwikCity(),
      qwikVite(),
      tsconfigPaths({ root: "." }),
      tailwindcss(),
    ],
    test: {
      exclude: ["e2e/**", "node_modules/**", "dist/**", ".rollup.cache/**"],
    },
    optimizeDeps: { exclude: [] },
    server: {
      headers: { "Cache-Control": "public, max-age=0" },
    },
    preview: {
      headers: { "Cache-Control": "public, max-age=600" },
    },
  };
});

// *** utils ***
// (función errorOnDuplicatesPkgDeps sin cambios)
```

**Espera — la lógica del adapter se aplica cuando `command === "build"`** vía un `extendConfig` separado. El base config queda como está; el adapter config se carga solo en build. Esto evita contaminar `pnpm dev` con la lógica del static adapter.

**Cómo se invoca el adapter en build:**

- En el Dockerfile: `pnpm build` (que ya invoca `qwik build`).
- El script `qwik build` de package.json probablemente ya hace lo correcto si detecta el adapter. **Verificar en `sdd-apply` que `qwik build` carga `adapters/static/vite.config.ts` automáticamente** (suele ser así por convención de Qwik). Si no, agregar `--mode static` o cambiar el script.

### 5.3 Rationale del static adapter

| Decisión | Alternativa | Por qué |
| ---------- | ------------- | --------- |
| Adapter escrito a mano | `pnpm qwik add static` | el `add` es interactivo en CI; la doc del adapter es pública |
| `origin: "http://localhost:3015"` | dejar vacío | necesario para que los `<link rel="canonical">` y OG tags se generen con el origin correcto |
| Sin `onStaticGenerate` | generar IDs concretos | las IDs son dinámicas (no se conocen en build); SPA fallback cubre el resto |
| Cargar el adapter solo en build | siempre cargado | `pnpm dev` no necesita el adapter; ahorra tiempo de cold start |

---

## 6. Refactor de las dos rutas

### 6.1 Diff de `frontend/src/routes/organizations/index.tsx`

**Antes** (routeLoader$ server-side):

```tsx
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import { OrganizationList, type OrganizationSummary } from "~/components/organization-list/organization-list";
import { listOrganizations } from "~/lib/api";

export const useOrganizationsLoader = routeLoader$(async () => {
  const result = await listOrganizations();
  if (result.ok) {
    const orgs: OrganizationSummary[] = result.value.map((o) => ({
      id: o.id,
      full_name: o.full_name,
      identification: o.identification,
    }));
    return { orgs };
  }
  return { orgs: [] as OrganizationSummary[], error: result.message };
});

export default component$(() => {
  const data = useOrganizationsLoader();
  return (
    <>
      {data.value.error && (
        <div role="alert" data-organization-list-error>{data.value.error}</div>
      )}
      <OrganizationList organizations={data.value.orgs} />
    </>
  );
});

// head sin cambios
```

**Después** (useVisibleTask$ client-side):

```tsx
import { component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";
import { OrganizationList, type OrganizationSummary } from "~/components/organization-list/organization-list";
import { listOrganizations } from "~/lib/api";

/**
 * /organizations — list or empty state.
 *
 * En SSG no hay server, así que la carga de datos se hace en el browser
 * después de hidratación. useVisibleTask$ corre SOLO en el client,
 * después de que el componente se vuelve visible.
 *
 * Empty arrays preservan el empty state (spec F-2). Transport failures
 * muestran [] más un banner de error (UX-3).
 */
export default component$(() => {
  const orgs = useSignal<OrganizationSummary[]>([]);
  const error = useSignal<string | null>(null);

  useVisibleTask$(async () => {
    const result = await listOrganizations();
    if (result.ok) {
      orgs.value = result.value.map((o) => ({
        id: o.id,
        full_name: o.full_name,
        identification: o.identification,
      }));
    } else {
      error.value = result.message;
    }
  });

  return (
    <>
      {error.value && (
        <div
          role="alert"
          class="mx-auto max-w-2xl border-b border-red-300 bg-red-50 px-4 py-2 text-sm text-red-900"
          data-organization-list-error
        >
          {error.value}
        </div>
      )}
      <OrganizationList organizations={orgs.value} />
    </>
  );
});

export const head: DocumentHead = {
  title: "Organizations · Cachicamas",
  meta: [
    { name: "description", content: "Browse and create organizations in Cachicamas." },
  ],
};
```

### 6.2 Diff de `frontend/src/routes/organizations/[id]/index.tsx`

**Antes** (routeLoader$ server-side):

```tsx
// (mismo patrón, con useOrganizationLoader que lee event.params.id)
```

**Después** (useVisibleTask$ + useLocation client-side):

```tsx
import { component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { type DocumentHead, useLocation } from "@builder.io/qwik-city";
import { OrganizationReadback, type OrganizationReadbackProps } from "~/components/organization-readback/organization-readback";
import { getOrganization } from "~/lib/api";

export default component$(() => {
  const loc = useLocation();
  const org = useSignal<OrganizationReadbackProps["organization"] | null>(null);
  const error = useSignal<string | null>(null);

  useVisibleTask$(async ({ track }) => {
    const id = Number(track(() => loc.params.id));
    if (!Number.isFinite(id) || id <= 0) {
      error.value = "Organization not found.";
      return;
    }
    const result = await getOrganization(id);
    if (result.ok) {
      org.value = {
        id: result.value.id,
        full_name: result.value.full_name,
        identification: result.value.identification,
      };
    } else {
      error.value = result.message;
    }
  });

  if (error.value || !org.value) {
    const message = error.value ?? "Organization not found.";
    return (
      <div class="mx-auto max-w-2xl space-y-4 px-4 py-8">
        <h1 class="text-3xl font-bold text-slate-900">Organization</h1>
        <div role="alert" data-organization-error class="…">{message}</div>
        <a href="/organizations" class="…">Back to organizations</a>
      </div>
    );
  }
  return <OrganizationReadback organization={org.value} />;
});

export const head: DocumentHead = {
  title: "Organization · Cachicamas",
  meta: [{ name: "description", content: "View a single organization in Cachicamas." }],
};
```

### 6.3 Diff de `frontend/src/routes/organizations/new/index.tsx` (NO CAMBIA)

`/organizations/new` sigue usando el mismo `$()` QRL para el submit. **No se toca**. La verificación e2e (crear org, navegar a readback) sigue funcionando porque el refactor solo afecta el readback, no el form.

### 6.4 Por qué `useVisibleTask$` y no `useTask$`

- `useTask$` corre en server y client. En SSG correría en build-time → datos congelados.
- `useVisibleTask$` corre **solo en el client**, después de hidratación. Es la elección correcta para "data que llega después del primer paint".
- Tradeoff: `useVisibleTask$` está marcado como "use with caution" en la doc de Qwik porque corre JS adicional. Para esta app es aceptable (es data loading, que es el caso de uso canónico).

### 6.5 Compatibilidad con specs Vitest existentes

Los specs (`src/routes/organizations/index.spec.tsx` y `src/routes/organizations/[id]/index.spec.tsx`) renderizan los **componentes presentacionales** (`OrganizationList`, `OrganizationReadback`) directamente, no las rutas. El refactor cambia el `default export` (loader → signal), pero el componente interno no cambia → los specs siguen pasando sin modificación.

---

## 7. docker-compose: servicio `frontend` y override file

### 7.1 Diff de `docker-compose.yaml` (servicio `frontend` agregado)

```yaml
  # … servicios existentes (postgres, jaeger, otel-collector, database_administrator) sin cambios …

  # --------------------------------------------------------------------------
  # frontend — Qwik static SSG served by nginx
  # --------------------------------------------------------------------------
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        # URL que el browser usa para hablar con el Go bin.
        # En dev local: http://localhost:8080 (puerto host del Go bin).
        # En VPS: depende de cómo se proxya (si se usa el override,
        #        nginx hace /api/* → database_administrator:8080,
        #        y PUBLIC_API_BASE_URL=/api).
        PUBLIC_API_BASE_URL: ${PUBLIC_API_BASE_URL:-/api}
    image: cachicamas/frontend:local
    container_name: cachicamas-frontend
    restart: unless-stopped
    ports:
      - "${FRONTEND_PORT:-3015}:80"
    networks:
      - cachicamas_network
    depends_on:
      database_administrator:
        condition: service_started
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://127.0.0.1/"]
      interval: 10s
      timeout: 5s
      retries: 6
      start_period: 10s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

**Nota**: el servicio `frontend` se publica con `${FRONTEND_PORT:-3015}:80`. En el override VPS, este mapping se mantiene (es el ÚNICO publicado).

### 7.2 Diff de `docker-compose.vps.yaml` (NUEVO — override)

```yaml
# ============================================================================
# Cachicamas — VPS override file.
#
# IMPORTANTE: este archivo redefine los servicios para deploy en VPS.
# Mantener sincronizado con docker-compose.yaml. Para validar que compone
# correctamente:
#   docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config
#
# Comportamiento:
#   - Solo el servicio `frontend` mantiene su `ports:` (3015:80).
#   - Los servicios internos (postgres, jaeger, otel-collector,
#     database_administrator) pierden su `ports:` → quedan en la red
#     privada.
#   - El Go bin se reconfigura con SERVICE_ENV=production (apaga el
#     default de CORS en dev) y CORS_ALLOW_ORIGINS (opt-in para casos
#     de borde).
#
# Uso:
#   docker compose -f docker-compose.yaml -f docker-compose.vps.yaml \
#     up -d --build
# ============================================================================

services:
  postgres:
    # Removemos ports: → el servicio queda accesible solo dentro de
    # cachicamas_network.
    ports: []

  jaeger:
    ports: []

  otel-collector:
    ports: []

  database_administrator:
    ports: []
    environment:
      # Reemplazamos el bloque de env del base file. Override completo
      # en compose v2 (no merge) — debemos re-listar todas las vars
      # o el servicio arranca sin ellas.
      # Para mantener el código del base file sin tocar, este override
      # reusa todas las env vars del base + agrega CORS_ALLOW_ORIGINS
      # y overridea SERVICE_ENV.
      # NOTA: en compose v2 los overrides REEMPLAZAN bloques; por
      # seguridad, el operador debe verificar el merge con `docker
      # compose config` antes de desplegar.
      SERVICE_NAME: database_administrator
      SERVICE_ENV: production  # override del default
      SERVICE_PORT: 8080
      POSTGRES_HOST: postgres
      POSTGRES_PORT: 5432
      POSTGRES_DB: ${POSTGRES_DB:-cachicamas_pg}
      POSTGRES_USER: ${QUEEN_USER:-queen}
      POSTGRES_PASSWORD: ${QUEEN_PASSWORD:-changeme-queen}
      OTEL_SERVICE_NAME: database_administrator
      OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4317
      OTEL_EXPORTER_OTLP_PROTOCOL: grpc
      OTEL_RESOURCE_ATTRIBUTES: service.namespace=${PROJECT_NAME:-cachicamas},deployment.environment=production
      CORS_ALLOW_ORIGINS: ${CORS_ALLOW_ORIGINS:-http://localhost:3015,https://cachicamas.example.com}

  # frontend no se redefine — su `ports:` del base file se mantiene
  # (es el único puerto publicado al host).
```

**Importante sobre overrides en compose v2**: los bloques se **reemplazan** completamente, no se mergean. Por eso el override del `database_administrator` tiene que re-listar **todas** las env vars, no solo agregar `CORS_ALLOW_ORIGINS`. Si no, el servicio arranca con un environment incompleto.

Alternativa evaluada: usar `extends:` o un merge config plugin. Descartado por simplicidad — re-listar las env vars es ~10 líneas, menos frágil que una dependencia.

### 7.3 Por qué override file, no `VPS_MODE` env var

| Approach | Pro | Contra |
| ---------- | ----- | -------- |
| **Override file** (elegido) | idiomático en compose v2; un revisor ve el stack VPS en un archivo; `docker compose config` valida el merge | dos archivos que mantener sincronizados |
| `VPS_MODE` env var con condicional | un solo archivo | YAML no tiene condicionales; profiles de compose no ocultan `ports:`; workaround con `x-` anchors es feo |
| Build de `docker-compose.yaml` con script | máxima flexibilidad | tooling adicional, debugging más difícil |

**Mecanismo de mantenimiento**: comment al tope de `docker-compose.vps.yaml` apuntando al base file. `sdd-verify` valida que el merge es correcto.

---

## 8. Refactor de `playwright.config.ts`

### 8.1 Diff de `frontend/playwright.config.ts`

**Antes**:

```ts
export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:5173",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  webServer: {
    command: "pnpm dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    stdout: "ignore",
    stderr: "pipe",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
```

**Después**:

```ts
const e2eBaseUrl = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const useDevServer = !process.env.E2E_BASE_URL;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: e2eBaseUrl,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  webServer: useDevServer
    ? {
        command: "pnpm dev",
        url: "http://localhost:5173",
        reuseExistingServer: !process.env.CI,
        timeout: 60_000,
        stdout: "ignore",
        stderr: "pipe",
      }
    : undefined,
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
```

### 8.2 Comportamiento

| `E2E_BASE_URL` | `baseURL` | `webServer` |
| ---------------- | ----------- | ------------- |
| (unset) | `http://localhost:5173` | levanta `pnpm dev` (dev local) |
| `http://localhost:3015` | idem | **undefined** (confía en el compose) |
| `https://cachicamas.example.com` | idem | **undefined** (VPS prod) |

### 8.3 Patch del spec e2e

En `frontend/e2e/create-organization.spec.ts`, agregar un `page.waitForLoadState("networkidle")` después de la navegación al readback. Esto espera a que el fetch client-side (`/api/organizations/{id}`) complete antes de los asserts.

```diff
   await page.goto("/organizations/new");
   await page.fill('input[name="full_name"]', unique("Acme"));
   // … resto del setup …
   await page.click('button[type="submit"]');
-  await expect(page).toHaveURL(/\/organizations\/\d+/);
+  await expect(page).toHaveURL(/\/organizations\/\d+/);
+  // Espera a que el fetch client-side de useVisibleTask$ complete.
+  await page.waitForLoadState("networkidle");
   await expect(page.getByRole("heading", { level: 1 })).toContainText("Acme");
```

**Por qué `waitForLoadState("networkidle")` y no un `expect().toBeVisible` específico**: la e2e actual busca contenido textual ("Acme"). El readback lo muestra cuando el fetch termina. `networkidle` es robusto y espera a que no haya requests en vuelo (incluyendo el fetch del Go bin).

---

## 9. Variables de entorno — diff de `.env.example`

### 9.1 Diff

```diff
 # ============================================================================
 # Cachicamas — local infrastructure env
 # ============================================================================

+# ─── Frontend (cachicamas-frontend-dockerize) ─────────────────────────────
+# Puerto host del frontend (mapea a :80 en el container nginx).
+# Default 3015. Para VPS, mantener 3015 o ajustar al puerto público deseado.
+FRONTEND_PORT=3015
+
+# URL que el browser usa para hablar con el Go bin.
+# - En dev local sin compose override: http://localhost:8080 (puerto host).
+# - En VPS con nginx reverse-proxy: /api (el nginx del frontend hace el proxy).
+# - En CI con compose: /api (el compose ya tiene nginx como reverse proxy).
+PUBLIC_API_BASE_URL=/api
+
+# Allowlist CORS del Go bin (separar origins por coma).
+# - Dev local default (cuando SERVICE_ENV=development): http://localhost:5173
+#   (configurado en el código de cors.go, no requiere env var).
+# - VPS: lista de origins válidos. Default conservador:
+#   http://localhost:3015,https://cachicamas.example.com
+#   Ajustar al dominio público real.
+CORS_ALLOW_ORIGINS=http://localhost:3015,https://cachicamas.example.com
+

 # ─── PostgreSQL (existente, sin cambios) ─────────────────────────────────
 POSTGRES_DB=cachicamas_pg
 POSTGRES_USER=cachicamas
 POSTGRES_PASSWORD=changeme-cachicamas  # CHANGE in production
 POSTGRES_PORT=5432
 POSTGRES_INITDB_ARGS=--encoding=UTF8
 QUEEN_PASSWORD=changeme-queen  # CHANGE in production

 # ─── Jaeger (existente) ─────────────────────────────────────────────────
 JAEGER_UI_PORT=16686
 JAEGER_OTLP_GRPC_PORT=4317
 JAEGER_OTLP_HTTP_PORT=4318

 # ─── OTel collector (existente) ─────────────────────────────────────────
 OTEL_COLLECTOR_GRPC_PORT=14317
 OTEL_COLLECTOR_HTTP_PORT=14318
 OTEL_COLLECTOR_HEALTH_PORT=13133

 # ─── database_administrator (existente) ─────────────────────────────────
 SERVICE_ENV=development
 SERVICE_PORT=8080
 PROJECT_NAME=cachicamas
```

### 9.2 Rationale

- `FRONTEND_PORT` separado de `SERVICE_PORT` (Go bin) y del resto: cada servicio tiene su variable.
- `PUBLIC_API_BASE_URL` con default `/api`: alineado con el diseño de nginx reverse-proxy. En dev local sin compose, se overridea a `http://localhost:8080`.
- `CORS_ALLOW_ORIGINS` con default permisivo para dev (`localhost:3015` y el dominio público de ejemplo): el operador puede ajustar al dominio real en producción.

---

## 10. Documentación — diff de `README.md` (raíz) y `frontend/README.md`

### 10.1 Diff de `README.md` (raíz, sección de deploy)

Agregar una sección corta (después de "Quick start"):

```diff
 ## Quick start

 [sección existente sin cambios]

+## Deploy to VPS
+
+El stack soporta un perfil "VPS" donde solo el frontend expone un puerto al
+host. Los servicios internos (Postgres, Go, Jaeger, OTel collector) quedan
+en la red privada `cachicamas_network`.
+
+```bash
+# Dev local (todos los servicios publicados):
+docker compose up -d --build
+
+# VPS (solo el frontend en :3015; el resto en la red privada):
+docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build
+```
+
+Para deploy en producción, ajustar `CORS_ALLOW_ORIGINS` en `.env` al
+dominio público real (`https://cachicamas.example.com`). El frontend
+queda accesible en `http://<host>:3015/`. El browser habla con el Go
+bin a través del reverse-proxy interno de nginx (`/api/*`).
```

### 10.2 Diff de `frontend/README.md` (sección "End-to-end tests")

```diff
 #### End-to-end tests

+Playwright corre contra el dev server por default. Para validar el stack
+dockerizado, usar `E2E_BASE_URL`:
+
+```bash
+# Dev local (default — Playwright arranca `pnpm dev` en 5173):
+pnpm test:e2e
+
+# Stack dockerizado (Playwright NO levanta webServer, confía en el compose):
+docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d
+E2E_BASE_URL=http://localhost:3015 pnpm test:e2e
+
+# VPS en producción (apunta al dominio público):
+E2E_BASE_URL=https://cachicamas.example.com pnpm test:e2e
+```

---

## 11. Riesgos específicos del diseño

| # | Risk (específico del diseño) | Mitigation |
|---|------------------------------|------------|
| D1 | El `qwik build` no detecta automáticamente `adapters/static/vite.config.ts` y no aplica el adapter | Verificar en `sdd-apply` que `qwik build` carga el adapter. Si no, cambiar el script `build` en `package.json` a `qwik build --mode static` o equivalente. |
| D2 | El override `database_administrator.environment` reemplaza todo el bloque, no merge. Si alguien edita el base y se olvida del override, el VPS arranca con env incompleto | Comment al tope del override file apuntando al base. `sdd-verify` corre `docker compose config` y verifica que el rendered config tiene todas las env vars esperadas. |
| D3 | `useVisibleTask$` puede disparar warnings de "Qwik prefers useTask$ for data loading" | El lint config del repo puede tener una regla sobre esto. Si la regla es estricta, usar `useTask$` con `track()` que retorne Promise, y documentar por qué la data no se prerenderiza. Por ahora, `useVisibleTask$` es la elección correcta. |
| D4 | nginx no tiene módulos preinstalados para `proxy_set_header Connection ""` o `gzip_types` extendido | La imagen `nginx:1.27-alpine` viene con los módulos core. `proxy_set_header` es core, `gzip` es core, `gzip_types` es core. Sin módulos extra requeridos. |
| D5 | El e2e con `E2E_BASE_URL` no maneja el caso "Playwright ve la página antes de que el JS bundle termine de hidratar" | El `waitForLoadState("networkidle")` post-navegación cubre esto (espera a que no haya requests en vuelo, lo que incluye la hidratación de Qwik y el fetch client-side). |
| D6 | `PUBLIC_API_BASE_URL=/api` en el build del frontend hace que el browser use la ruta relativa. Si el usuario entra a `http://localhost:3015/organizations` (sin trailing slash) y el form hace `fetch('/api/organizations', ...)`, el browser lo resuelve a `http://localhost:3015/api/organizations` — correcto. Pero si el usuario entra a `http://localhost:3015` (root), y hay un `<a href="/api/...">` en el HTML, el browser también lo resuelve a `http://localhost:3015/api/...` — correcto. Sin embargo, si el frontend se sirve bajo un subpath (e.g., `http://example.com/app/`), el default `/api` no funcionaría. Aceptable: el design asume root-path. | Documentar en README que el deploy asume root-path. Si en el futuro se necesita subpath, ajustar `PUBLIC_API_BASE_URL` y `nginx.location /`. |

---

## 12. Forecast del budget y chained PR

| Archivo | Tipo | Líneas estimadas (add+del) |
|---------|------|----------------------------|
| `frontend/Dockerfile` | New | ~50 |
| `frontend/.dockerignore` | New | ~15 |
| `frontend/nginx.conf` | New | ~75 |
| `frontend/adapters/static/vite.config.ts` | New | ~25 |
| `frontend/vite.config.ts` | Modified | +5 (registro del adapter) |
| `frontend/src/routes/organizations/index.tsx` | Modified | ~10 net change (loader → signal) |
| `frontend/src/routes/organizations/[id]/index.tsx` | Modified | ~20 net change (loader + useLocation → signal) |
| `frontend/playwright.config.ts` | Modified | ~10 net change (env-driven) |
| `frontend/e2e/create-organization.spec.ts` | Modified | +3 (waitForLoadState) |
| `docker-compose.yaml` | Modified | +30 (servicio frontend) |
| `docker-compose.vps.yaml` | New | ~70 |
| `.env.example` | Modified | +15 (3 variables nuevas) |
| `README.md` (raíz) | Modified | +20 (sección deploy) |
| `frontend/README.md` | Modified | +10 (sección e2e) |
| **TOTAL** | — | **~358 líneas** |

**Forecast**: **Medium** (cerca del límite 400). **Single PR recomendado** (no chained).

**Riesgo de revisión**: 14 archivos, 13 modificaciones de archivos existentes. La superficie de código de aplicación es mínima (2 rutas + 1 spec + 1 config); el resto es config/infra/docs. Un revisor puede auditar el change en < 30 min.

**Si el forecast pasara 400**: chained PR en 2:
- PR1: Dockerfile + nginx + adapters (infra).
- PR2: refactor rutas + e2e + compose + override (código + integración).

Por ahora, single PR.

---

## 13. Resultado

- **Status**: ok
- **Executive summary**: 12 secciones de diseño que aterrizan todas las decisiones del proposal y los specs. 6 archivos nuevos (Dockerfile, .dockerignore, nginx.conf, adapter, override, …) + 7 archivos modificados (rutas, vite.config, playwright, compose, .env, READMEs). Forecast: ~358 líneas, single PR.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/design.md` (este archivo), engram topic `sdd/cachicamas-frontend-dockerize/design`.
- **Next recommended**: `sdd-tasks` (romper el design en tareas de implementación).
- **Skill resolution**: `paths-injected`.
