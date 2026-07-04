# syntax=docker/dockerfile:1.7
#
# mocks-github-oauth — minimal Node service that simulates the GitHub
# OAuth + /user + /user/emails endpoints for the cachicamas-github-login
# PR-2 Playwright e2e suite.
#
# Stage 1: builder
#   - node:22-alpine (matches the rest of the project's runtime).
#   - pnpm install for self-doc consistency (we have no deps; pnpm just
#     produces a deterministic node_modules tree).
#
# Stage 2: runner
#   - node:22-alpine (slim).
#   - COPY only the server script + package.json.
#   - HEALTHCHECK via /healthz.
#
# Why this lives in scripts/ (not infra/):
#   It's a test-only fixture. Production never sees this image. Tagging
#   it `cachicamas/mocks-github-oauth:local` keeps the convention
#   (every compose service has a `cachicamas/<name>:local` tag) without
#   implying it's deployable.

FROM node:22-alpine AS builder

WORKDIR /app

# Self-doc: even though we have zero runtime deps, having a package.json
# lets the compose healthcheck pass metadata about the service.
COPY package.json ./

# Stage 2: runner (no deps to install).
FROM node:22-alpine AS runner

WORKDIR /app

COPY --from=builder /app/package.json ./package.json
COPY server.mjs ./server.mjs

ENV MOCKS_HOST=0.0.0.0
ENV MOCKS_PORT=3016

EXPOSE 3016

HEALTHCHECK --interval=10s --timeout=5s --retries=6 --start-period=5s \
  CMD node -e "require('http').get('http://127.0.0.1:'+process.env.MOCKS_PORT+'/healthz',r=>process.exit(r.statusCode===200?0:1)).on('error',()=>process.exit(1))"

CMD ["node", "server.mjs"]