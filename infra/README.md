# Cachicamas — Infrastructure

Local-development infrastructure for the cachicamas platform, packaged as
Docker Compose services.

## Topology

```
                ┌─────────────────────┐
                │ database_admin      │  (Go 1.26 microservice)
                │ :8080  (host)       │
                └─────────┬───────────┘
                          │  OTLP/gRPC :4317  (in-network)
                          ▼
                ┌─────────────────────┐
                │  otel-collector     │  (gateway: receives, batches,
                │  :13133 health      │   exports)
                │  host: 14317/14318  │
                └─────────┬───────────┘
                          │  OTLP/gRPC :4317  (in-network)
                          ▼
                ┌─────────────────────┐
                │      jaeger         │  (v2 / all-in-one, in-memory store)
                │  :16686 UI          │
                │  host: 4317/4318    │
                └─────────────────────┘

                ┌─────────────────────┐
                │     postgres        │  (18-alpine, persistent volume)
                │  :5432              │
                └─────────────────────┘
```

The microservice has **no direct dependency on Jaeger** — it only knows
about the OTel Collector (via the in-network DNS name `otel-collector:4317`).
Swap the exporter in `infra/otel/collector-config.yaml` and the same
image runs in production against any OTLP-compatible backend.

> **Why the host-port offset?** Both Jaeger and the OTel Collector expose
> OTLP on container-port `4317`. Docker Desktop's vpnkit sidecar holds a
> stale `4317` host-bind across container restarts. To avoid the collision
> we publish the Collector to `14317/14318` on the host while keeping the
> standard `4317/4318` *inside* the container network.

## Services

| Service              | Image                                       | Host port(s)        | Purpose                                  |
|----------------------|---------------------------------------------|---------------------|------------------------------------------|
| `postgres`           | `postgres:18-alpine`                        | `5432`              | Primary database                         |
| `otel-collector`     | `otel/opentelemetry-collector-contrib:0.137.0` | `14317`, `14318`, `13133` | Telemetry gateway (offset host-ports — see topology note) |
| `jaeger`             | `jaegertracing/jaeger:2.0.0`                | `16686`             | Distributed-tracing UI                   |
| `database_administrator` | built from `backend/database_administrator/Dockerfile` | `8080` | The microservice |

## Quick start

```bash
# 1. Copy the environment template and edit credentials.
cp .env.example .env

# 2. Bring everything up (builds the Go service from local source).
#    Note: if Docker Desktop has stale port-binds from a previous run,
#    quit Docker Desktop (Cmd+Q) and reopen it before this step.
docker compose up -d --build

# 3. Watch the microservice logs while you hit it.
docker compose logs -f database_administrator

# 4. Smoke-test the health endpoint.
curl -i http://localhost:8080/health

# 5. Open the Jaeger UI and pick `database_administrator` from the
#    "Service" dropdown — you should see the request as a span.
open http://localhost:16686

# 6. (Optional) Send a test OTLP payload directly to the Collector on
#    the host. Note the offset port (14317, not 4317).
grpcurl -plaintext -d '{"resource_spans":[]}' \
    localhost:14317 otlp.collector.metrics.v1.ExportMetricsService/Export || true
```

## Database roles

Two roles exist after the first `docker compose up`:

| Role          | Type             | Provisioned by | Used for                                              |
|---------------|------------------|----------------|-------------------------------------------------------|
| `cachicamas`  | SUPERUSER        | official image | Bootstrap. Not used by the app or by you in practice. |
| `queen`       | DBA (nosuper)    | `01-init.sql`  | Migrations, DDL, provisioning new roles & databases.  |

`queen` has `CREATEROLE + CREATEDB + REPLICATION` but is **not** a
superuser. Future per-context roles (e.g. `wiki`, or a least-privilege
`cachicamas_app`) are created by running SQL as `queen` — a recipe is
commented at the bottom of `01-init.sql`. Until then, microservices also
connect as `queen` because it's the only non-superuser role that exists.

Password comes from `QUEEN_PASSWORD` in `.env`; the default is dev-only
and **must** be replaced in any real env.

## Where to look

| What                  | Where                                                                 |
|-----------------------|-----------------------------------------------------------------------|
| Postgres logs         | `docker compose logs -f postgres`                                      |
| Postgres data         | `data/postgres/` (named volume, persists across `up`/`down`)           |
| Telemetry pipeline    | `docker compose logs -f otel-collector`                               |
| Spans (live UI)       | <http://localhost:16686>                                               |
| Collector self-health | <http://localhost:13133>                                               |
| zPages (pipeline viz) | <http://localhost:55679/debug/tracez>                                  |
| Microservice          | `http://localhost:8080`                                                |

## Operations

```bash
# Stop everything (keeps volumes).
docker compose down

# Stop AND wipe data (irreversible).
docker compose down -v

# Rebuild only the Go service after a code change.
docker compose up -d --build database_administrator

# Tail Jaeger logs to see incoming spans.
docker compose logs -f jaeger

# Open a psql shell inside the postgres container.
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"

# Validate the collector config without restarting.
docker compose exec otel-collector \
  otelcol-contrib validate --config=/etc/otelcol/config.yaml
```

## When you move to production

1. Replace Jaeger `all-in-one` with a real backend (Tempo, Elasticsearch, …).
   Change `infra/otel/collector-config.yaml` `exporters.otlp/jaeger` to point
   at it. Application code does not change.
2. Move Postgres off the named volume and onto a managed service
   (RDS, Cloud SQL, Neon, Supabase). Update `POSTGRES_HOST`.
3. Switch the microservice runtime image from `gcr.io/distroless/static` to
   your registry (`ghcr.io/cachicamas/database_administrator:<sha>`).
4. Remove the `debug` exporter from the collector pipelines — it prints to
   stdout and produces noise.
5. Set real resource attributes
   (`OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production,...`).