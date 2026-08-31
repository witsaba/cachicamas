# syntax=docker/dockerfile:1.7
#
# Cachicamas frontend — multi-stage build con Node SSR.
#
# Stage 1: builder
#   - node:22-alpine
#   - pnpm install (deps completas, incluido devDeps para Vite)
#   - pnpm build → genera dist/ (chunks client) + server/ (bundle Node SSR)
#
# Stage 2: runner
#   - node:22-alpine (mismo base, sin nginx ni extras)
#   - instala SOLO production deps del frontend (más liviano)
#   - copia dist/ (chunks) + server/ (Node SSR) + package.json
#   - entry point: `node server/entry.express.js`
#
# Output esperado: ~80MB final image.
#
# ¿Por qué Node SSR y no Static SSG?
#   El app tiene rutas dinámicas (/organizations/{id}/) con IDs creados
#   en runtime por el form submit. Static SSG no puede prerenderizar IDs
#   arbitrarios — solo IDs fijos via onStaticGenerate. Node SSR ejecuta
#   el route loader en cada request, así que orgs recién creadas son
#   inmediatamente legibles.
#
# Reverse proxy /api/* → http://database_administrator:8080 está
# implementado en src/entry.express.tsx (Node http server con proxy
# middleware). No se necesita nginx.

# ─── Stage 1: builder ────────────────────────────────────────────────────
FROM node:22-alpine AS builder

# corepack ships con node:22-alpine; respeta el `packageManager` del package.json.
RUN corepack enable

WORKDIR /app

# CI=true disables pnpm's TTY-only confirmations for non-interactive builds.
ENV CI=true

# Copiamos solo los manifests primero para aprovechar el cache de Docker.
# .npmrc MUST come along so pnpm reads shamefully-hoist=true before install.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
RUN pnpm install --frozen-lockfile --shamefully-hoist

# Recién ahora copiamos el resto.
COPY . .

# Build arg para PUBLIC_API_BASE_URL (Vite lo inlinea en build-time).
# Default: /api (relativo — el Qwik server tiene /api/* proxy al Go bin).
ARG PUBLIC_API_BASE_URL=/api
ENV PUBLIC_API_BASE_URL=${PUBLIC_API_BASE_URL}

# Origen canónico (Qwik lo usa para canonical/OG tags y CSRF validation).
# Default: el public-facing URL del VPS. Override en docker-compose.
ARG ORIGIN=http://localhost:3015
ENV ORIGIN=${ORIGIN}

# Genera el client bundle + el server bundle (SSR + reverse proxy).
RUN CI=true pnpm build

# ─── Stage 2: runner ────────────────────────────────────────────────────
FROM node:22-alpine AS runner

WORKDIR /app

# El server bundle de Qwik (vite) tiene la mayoría de las deps bundleadas
# (zod, etc.) como imports estáticos resueltos por Vite. Pero hay algunas
# deps que se importan dinámicamente (e.g., `undici` para fetch/HttpRequest)
# y no se bundlean. Las copiamos desde el builder stage (donde pnpm
# instaló todo el árbol de deps).
COPY --from=builder /app/node_modules ./node_modules

# Copiamos los artefactos del builder.
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/server ./server

# Variables de entorno por defecto. docker-compose puede overridear.
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000
# /api/* reverse proxy target. docker-compose overridea al service name.
ENV API_TARGET=http://database_administrator:8080
# Browser uses PUBLIC_API_BASE_URL (relative /api goes via nginx).
# Server uses SERVER_API_BASE_URL (absolute, in compose network).
ENV PUBLIC_API_BASE_URL=/api
ENV SERVER_API_BASE_URL=http://database_administrator:8080
ENV ORIGIN=http://localhost:3015

EXPOSE 3000

# Healthcheck via Node: hace un fetch al endpoint raíz. No requiere
# wget/curl (la imagen `node:22-alpine` no los trae por default en
# el slim layout, y no queremos instalar más paquetes).
HEALTHCHECK --interval=10s --timeout=5s --retries=6 --start-period=10s \
  CMD node -e "require('http').get('http://127.0.0.1:'+process.env.PORT+'/',r=>process.exit(r.statusCode<400?0:1)).on('error',()=>process.exit(1))"

# El server lee PORT y HOST del env. CMD explícito para evitar drift.
CMD ["node", "server/entry.express.js"]