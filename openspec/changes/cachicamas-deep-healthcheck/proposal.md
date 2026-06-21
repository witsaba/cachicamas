# Proposal: cachicamas-deep-healthcheck

> **Cambio**: `cachicamas-deep-healthcheck`
> **Status**: proposed
> **Created**: 2026-06-21
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-deep-healthcheck/proposal`)

---

## Intent

Hoy, el endpoint `GET /health` de `database_administrator` siempre devuelve HTTP 200 con `{"status":"ok"}` — **no consulta la base de datos**. Esto fue evidente el 2026-06-21 cuando intentamos verificar el sampler del `otel-collector` parando postgres: los 10 requests a `/health` devolvieron 200 sin error.

Esto significa:

- **El healthcheck miente**: el load balancer / orquestador piensa que el servicio está "vivo y útil" cuando la base de datos podría estar caída y todas las requests reales fallando.
- **No se puede detectar degradación** en dependencias externas (Postgres hoy; mañana Redis, otros servicios).
- **Falla el principio de fail-fast**: un deploy con DSN mal configurado o permisos rotos solo se descubre cuando un cliente pega al endpoint real.
- **No cumple prácticas SRE modernas**: Kubernetes distingue entre liveness (¿proceso vivo?) y readiness (¿proceso puede servir tráfico útil?). Hoy tenemos un solo endpoint que mezcla ambos conceptos y no chequea nada.

Este cambio introduce el patrón **liveness + readiness separados** con verificación real de la base de datos en readiness.

## Scope

### In Scope

- **Dos endpoints HTTP distintos**:
  - `GET /livez` — liveness. Devuelve 200 si el proceso Go está vivo y el HTTP server responde. **No** consulta la base de datos. No causa restart por dependencias caídas.
  - `GET /readyz` — readiness. Devuelve 200 si el proceso + la conexión a Postgres están sanos. Devuelve 503 si el `db.PingContext` falla o el pool está exhausto. Causa que el load balancer saque el pod del pool, sin reiniciarlo.
- **Conexión real a Postgres** usando `pgx/v5` (driver nativo Go, mejor que `lib/pq` que está deprecado) o `database/sql + pgx stdlib`.
- **`db.PingContext(ctx)` con timeout** de 2s para el check, llamado solo en `/readyz` (no en cada request de negocio).
- **Remover el `?fail=true` flag dev-only** que agregamos al handler de `/health` en el PR de `cachicamas-tail-sampling`. Una vez que `/readyz` está separado de `/livez`, el `?fail=true` ya no es necesario para tests (los tests 3.1 pueden usar `/livez` para happy path y `/readyz` con un DSN inválido para forzar fallo).
- **`/health` se mantiene como alias de `/readyz`** por un período de gracia, con un header `Deprecation: true` o un log de warning. Decisión fina se toma en el design phase.
- **Configuración**: leer `POSTGRES_*` env vars (ya en uso en `docker-compose.yaml`) y construir el pool con `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime` razonables para dev.
- **Inicialización ordenada**: el servidor HTTP no abre el puerto `:8080` hasta que `db.PingContext` con un timeout de 30s haya pasado. Esto previene que el pod reciba tráfico antes de estar listo (cascading failures en arranque).
- **Documentación**: actualizar el comentario del handler, agregar una sección en el README (cuando exista) sobre los dos endpoints, y dejar el `?fail=true` flag documentado en el código como removible.

### Out of Scope (deferred pero relacionado)

- **Liveness + readiness en formato Kubernetes (probes)**: este proyecto usa docker-compose, no k8s. Cuando migramos a k8s (otro SDD change), el `docker-compose` se reemplaza y los probes se configuran ahí.
- **HTTP `/startupz` separado**: no se justifica para un servicio que arranca en <5s. Se documenta como follow-up.
- **Circuit breakers en checks de readiness**: si Postgres se cae, el `PingContext` va a fallar consistentemente; el circuit breaker es para no martillar la DB con checks. Hoy un check cada 5s (frecuencia típica de readiness probe) no es problema. Dejar para cuando agreguemos más dependencias.
- **Authentication/authorization en health endpoints**: por convención, los health endpoints son públicos (mismos ejemplos: Spring Actuator, k8s probes). Documentar como decisión de diseño, no agregar.
- **Cache de resultados de PingContext**: con 1 check cada 5s no hace falta. Dejar para cuando haya 5+ dependencias.
- **Cambiar el driver de Postgres (`lib/pq` → `pgx/v5`)**: si resulta que `database/sql + lib/pq` es suficiente para `db.PingContext`, no migramos. Decisión en design phase.

## Approach

### High-level

```
                ┌─────────────────────────────────────────┐
   GET /livez ─►│  LivenessHandler                        │
                │  - ¿Proceso vivo? Sí → 200              │
                │  - No toca la DB                        │
                └─────────────────────────────────────────┘

                ┌─────────────────────────────────────────┐
   GET /readyz ►│  ReadinessHandler                       │
                │  1. ¿Startup completo? No → 503         │
                │  2. ¿db.PingContext(2s) OK? No → 503     │
                │  3. ¿Pool stats sano? No → 503           │
                │  4. Todo bien → 200 con detalle JSON     │
                └─────────────────────────────────────────┘
```

### Stack técnico (decisión tentativa, refinar en design)

- **Driver**: `pgx/v5` con `stdlib` adapter (`pgx/v5/stdlib`) para usar `database/sql` estándar. Permite `db.PingContext` con el `pgx` driver por debajo. Si `pgx` resulta problemático en Alpine/distroless, fallback a `lib/pq`.
- **Sin nuevos frameworks**: no引入 healthcheck libs (`go-kit`, `heptio/healthcheck`). El patrón es ~80 LoC y no merece una dep.
- **No introduce arquitectura hexagonal adicional**: el `*sql.DB` se inyecta al constructor de `HealthService` igual que hoy se inyecta el servicio. La aplicación sigue siendo el orquestador.

### Archivos a tocar (estimación, refinar en design)

| Archivo | Cambio | LoC estimado |
|---|---|---|
| `backend/database_administrator/go.mod` | Agregar `github.com/jackc/pgx/v5` | +3 |
| `backend/database_administrator/src/application/health_service.go` | Recibir `*sql.DB`, agregar `CheckDependencies(ctx) error` | +15 |
| `backend/database_administrator/src/interfaces/http/liveness_handler.go` (NEW) | Handler simple, 200 always | ~20 |
| `backend/database_administrator/src/interfaces/http/readiness_handler.go` (NEW) | Handler con PingContext + pool stats | ~60 |
| `backend/database_administrator/src/interfaces/http/health_handler.go` | Remover `?fail=true` flag, mantener como alias a readiness | -10 |
| `backend/database_administrator/src/cmd/server/main.go` | Inicializar `*sql.DB` antes de abrir HTTP, inyectar al HealthService | +25 |
| `backend/database_administrator/Dockerfile` | Sin cambios (la nueva dep entra en go.mod/go.sum) | 0 |
| `docker-compose.yaml` | Sin cambios (las env vars ya existen) | 0 |
| Tests (unit) | Testear los 3 caminos: ready, no-ready por DB down, no-ready por pool exhausto | ~120 |

**Total estimado**: ~230 LoC Go + 3 LoC go.mod = **~233 LoC**, dentro del budget 400.

### Rollback

- `git revert <merge-commit>` + `docker compose up -d --build`.
- `/health` sigue funcionando como antes (alias a `/readyz`).
- Si el revert causa un crash en el startup porque `db.PingContext` falla antes de abrir el puerto, abrir el puerto de todos modos (es la versión actual, "always healthy" mentirosa) y degradar con un log de error.

## Out of Scope (estricto, NO en este cambio)

- **No tocamos el OTel collector config** — el sampler ya está merged/merge-pending.
- **No tocamos docker-compose.yaml** — las env vars necesarias ya están.
- **No tocamos tests de integración del sampler** — siguen funcionando con el `?fail=true` flag mientras conviva con `/readyz`, o se migran en un follow-up trivial.
- **No agregamos Prometheus metrics del health check** — se puede hacer en un SDD aparte sobre métricas.

## Non-goals

- No vamos a hacer un "service mesh-aware" health check que considere el estado de otros servicios en el cluster. La práctica recomendada (Henning Jacobs / Zalando) es **evitar** esa dependencia para no causar cascading restarts.
- No vamos a usar `exec` probes en el docker-compose healthcheck (ya documentado por qué: distroless/static no tiene shell).
- No vamos a hacer health checks condicionales basados en `SERVICE_ENV`. La lógica es la misma en dev y prod — el fail-injection se elimina de una vez, no se gatea.

## Open Questions (a resolver en design phase)

1. ¿`pgx/v5` puro (más performante) o `pgx/v5/stdlib` (compatible con `database/sql` y `PingContext`)? Recomendación inicial: `stdlib` por simplicidad, migrar a puro si hay benchmark que lo justifique.
2. ¿Cuántas conexiones en el pool para dev? Sugerido: `MaxOpen=10`, `MaxIdle=2`, `ConnMaxLifetime=5min`. Validar en stress test simple.
3. ¿`/health` como alias de `/readyz` durante 1 release, o se rompe la compatibilidad de una? Sugerido: alias por 1 release, log warning si se usa.
4. ¿Qué nivel de detalle en la respuesta JSON de `/readyz`? Sugerido: `{ "status": "ok", "checks": { "postgres": { "status": "ok", "latency_ms": 3 } } }` cuando está healthy; `{ "status": "unhealthy", "checks": { "postgres": { "status": "fail", "error": "dial tcp 10.0.0.5:5432: i/o timeout" } } }` cuando falla.
5. ¿Log cada transición de healthy↔unhealthy, o solo en unhealthy? Sugerido: log transiciones (cambio de estado) para no inundar logs en estado estable.

## Verification

- Unit tests del handler (con `*sql.DB` mockeado o `sqlmock`): 3 casos (healthy, ping fail, pool exhausto).
- Integration contra compose: `curl /readyz` con DB arriba → 200; parar postgres → 503; reiniciar postgres → 200 (recovery en <2s).
- Contrato con el sampler: el comportamiento NO debe cambiar (sigue emitiendo spans de los 2 endpoints).

## References

- Investigado en web 2026-06-21: [Kubernetes probes docs](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), [Google Cloud: best practices for health checks](https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-setting-up-health-checks-with-readiness-and-liveness-probes), [Henning Jacobs (Zalando): Liveness Probes are Dangerous](https://srcco.de/posts/kubernetes-liveness-probes-are-dangerous.html), [Spring Blog: Liveness and Readiness Probes with Spring Boot](https://spring.io/blog/2020/03/25/liveness-and-readiness-probes-with-spring-boot), [OneUptime: Health Checks in Go for Kubernetes](https://oneuptime.com/blog/post/2026-01-07-go-health-checks-kubernetes/view), [Nick Janetakis: Splitting Web App Health Check URLs](https://nickjanetakis.com/blog/splitting-out-web-app-health-check-urls-for-basic-and-database-checks), [Theo "Bob" Massard: Complete Health Checks and Why They Matter](https://medium.com/@tbobm/complete-health-checks-and-why-they-matter-8b2120d86e4f).
- Conexión con PR previo: `cachicamas-tail-sampling` (PR #2, en cola). El `?fail=true` flag que agregamos a `/health` en ese PR se elimina en este change.
