# Proposal: cachicamas-tail-sampling

> **Cambio**: `cachicamas-tail-sampling`
> **Status**: proposed
> **Created**: 2026-06-20
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (este archivo + engram `sdd/cachicamas-tail-sampling/proposal`)

---

## Intent

Hoy, la aplicación `database_administrator` envía **el 100% de los traces** al otel-collector, que los reenvía al 100% a Jaeger (`traces` pipeline: `otlp → batch → otlp/jaeger + debug`). El `debug` exporter muestrea 1 de cada 200 después del primero para no inundar stdout, pero el exporter a Jaeger no muestrea — todo pasa.

Esto es aceptable mientras el tráfico sea bajo. Cuando entren en juego más servicios o más carga:

- **Se pierde señal** de los errores y latencias altas (porque se guardan mezclados con el 100% del tráfico feliz y se descartan visualmente en la UI).
- **Costo de storage** en Jaeger crece linealmente con el tráfico sin importar si los traces son interesantes.
- **No se puede hacer fan-out** ("errores a un lado, exitosos a otro") sin esta pieza.

**Este cambio introduce tail-based sampling en el otel-collector**: el collector espera a que cada trace se complete, mira atributos (errores, latencia, status codes) y decide si lo exporta a Jaeger o lo descarta. Resultado: 100% de errores y latencias altas guardados, 1–5% del tráfico feliz guardado, con un solo cambio de config.

## Scope

### In Scope

- Agregar el processor `tail_sampling` a `infra/otel/collector-config.yaml` en el pipeline `traces`.
- Política de muestreo compuesta (combinators `and` / `or`):
  - **OR high-value (100% keep)**: error status, exception, latency > 1s, HTTP 5xx, gRPC `code != OK`.
  - **OR tráfico feliz (1–5% probabilistic keep)**: requests exitosos muestreados probabilísticamente.
  - **Política catch-all** (5% probabilistic) para spans que no matchean ninguna.
- Load balancing extension (`loadbalancing_exporter`) **NO** incluida — sigue siendo single-node.
- Carga de memoria del collector: ajustar `memory_limiter.limit_percentage` y documentar el límite.
- Cargar el sampler con `decision_cache` y `num_traces` dimensionados para el tráfico local esperado (decenas de miles de spans/min).
- Pruebas de verificación: comparar volumen antes/después en Jaeger; confirmar que un error 500 sigue presente al 100% en la UI.
- Documentar en `infra/otel/collector-config.yaml` cada policy con comentario inline explicando qué captura y por qué.

### Out of Scope (deferred pero relacionado)

- **Logs persistentes** (Loki / file) — la pipeline `logs` sigue yendo a `debug` (stdout) y nada más. Diferente PR.
- **Métricas persistentes** (Prometheus exporter + scraper) — la pipeline `metrics` también sigue yendo a `debug`. Diferente PR.
- **File storage / queue persistente** del collector — el `sending_queue` de 5000 spans en memoria sigue igual. Diferente PR si se justifica.
- **Tail sampling load balancing** (multi-collector con `loadbalancing_exporter` + consistent hashing) — no aplica a single-node.
- **Self-metrics del collector** expuestos en un endpoint — sigue in-process.
- **Cambios en la app Go** — el SDK no necesita tocarse. Esta propuesta es 100% config del collector.

## Capabilities

### New Capabilities

- `trace-tail-sampling`: Capacidad del otel-collector de cachicamas para decidir post-hoc (después de ver el trace completo) qué traces exportar a Jaeger, basándose en errores, latencia, status codes, y muestreo probabilístico del tráfico feliz. Cubre requirements para escenarios Given/When/Then en la fase `sdd-spec`.

### Modified Capabilities

- **None**. Ningún spec existente cambia a nivel de requirements. El `sdd-spec` phase creará un único spec nuevo (`trace-tail-sampling/spec.md`); no hay delta specs sobre capabilities existentes.

## Approach

**Topología actual** (sin cambios en la topología física):

```
app (Go) ──OTLP/gRPC──> otel-collector ──OTLP──> jaeger
                                       └─debug──> stdout
```

**Topología propuesta** (cambia solo la pipeline `traces` dentro del collector):

```
app (Go) ──OTLP/gRPC──> otel-collector ──[tail_sampling]──> batch ──> otlp/jaeger
                                                              └─debug (mantenido)
```

Pipeline `traces` nueva:

```yaml
receivers:  [otlp]
processors: [memory_limiter, resourcedetection, resource, tail_sampling, batch]
exporters:  [otlp/jaeger, debug]
```

El processor `tail_sampling` se inserta **antes** de `batch` (es la posición canónica: el sampler necesita ver spans en vuelo para armar la decisión por trace; el batcher comprime después).

**Políticas (resumen de orden de evaluación):**

1. `errors` (composite: status_code=ERROR OR exception event exists) → keep
2. `slow` (composite: latency > 1000ms) → keep
3. `http5xx` (composite: http.response.status_code class = 5xx) → keep
4. `grpc_failures` (composite: rpc.grpc.status_code != OK) → keep
5. `probabilistic_happy` (5% rate, attribute filter: no error, latency < 1s) → keep
6. `catch_all` (1% rate, fallback) → keep

Las políticas 1–4 son OR-ed (cualquier condición = keep, no todas a la vez), 5 y 6 son OR-ed con las políticas 1–4 (cualquier match en cualquier política retiene el trace), y la política final catch-all asegura que algo siempre se exporta si ninguna otra matcheó (evita "0 traces" en silencio).

**Memoria esperada del collector:** con `num_traces: 50000` y `decision_cache` en memoria, el RSS del collector se mueve de ~80MB a ~200–300MB local. Aceptable para un dev stack. El `memory_limiter` se baja de 80% a 60% para dejar headroom.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `infra/otel/collector-config.yaml` | Modified | Agregar `tail_sampling` processor; reordenar pipeline `traces`; ajustar `memory_limiter`; documentar cada policy con comentario. |
| `docker-compose.yaml` | No change | Imagen `otel/opentelemetry-collector-contrib:0.137.0` ya incluye el `tail_sampling` processor en el build contrib (no se requiere cambio de versión). |
| `backend/database_administrator/**` | No change | El SDK OTel no necesita tocarse. Sigue mandando al `OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4317` como hasta ahora. |
| `infra/jaeger/all-in-one.yaml` | No change | Jaeger recibe menos traces pero el contrato OTLP no cambia. |
| `README.md` | Modified | Una nota corta (3-5 líneas) en la sección de observabilidad explicando "Jaeger ahora guarda ~5% del tráfico feliz y 100% de errores/lentos". |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| El `tail_sampling` processor aumenta el uso de memoria del collector → OOM kill en dev | Medium | Ajustar `memory_limiter.limit_percentage` de 80 a 60; documentar la cota esperada (200–300MB); monitorear con `docker stats` durante verify. Si el host tiene < 4GB RAM total, el dev stack se ajusta. |
| Las policies son demasiado agresivas y se pierden errores raros (no clasificados en 1–4) | Low | Catch-all policy al 1% asegura que SIEMPRE hay una muestra del tráfico no clasificado. Errores raros con 1% sampling se ven ~3 veces por hora con tráfico de 1k req/min. Aceptable para dev. |
| Las policies son demasiado permisivas y Jaeger recibe casi todo | Low | Verificación obligatoria en `sdd-verify`: comparar volumen Jaeger antes/después. Si la reducción es < 50%, ajustar `probabilistic_happy.rate` o las policies compuestas. |
| Latencia del sampler agrega overhead al pipeline (el trace espera a completarse antes de exportar) | Low | El sampler evalúa cuando llega el `root span` end o expira el `decision_wait`. `decision_wait: 10s` es el default; explícito en config. Para dev, ese delay es invisible. |
| Conflicto con `debug` exporter: el debug muestra 1/200 hoy; con tail sampling puede mostrar 0 si el sampler decide descartar antes | Low | `debug` queda como exporter final del pipeline, recibe lo mismo que `otlp/jaeger`. Si queremos mantener visibilidad total para debug local, podemos agregar un exporter paralelo fuera del sampler (pipeline separada). Decisión: hacerlo solo si `sdd-verify` lo justifica. |
| Cambio de comportamiento rompe asunciones de testing/CI existente | Low | Los tests de la app Go no tocan el pipeline del collector. El healthcheck del compose (`/otelcol-contrib validate`) valida la config antes de aceptar el container. |

## Rollback Plan

1. **Reversión inmediata** (en caso de incidente): `git revert <commit-hash>` del PR, `docker compose up -d --build` para reiniciar el collector con la config previa.
2. **Sin cambios de schema** en Jaeger (es OTLP estándar, el collector sigue siendo el source of truth).
3. **Sin cambios en la app Go** → cero riesgo de regresión funcional en el servicio.
4. **Fallback operacional** (si revert no es opción inmediata): comentar la línea `tail_sampling` en el array `processors` de la pipeline `traces` y reiniciar. Esto bypasea el sampler, vuelve a head-based 100% export. Tiempo de ejecución: < 30s.
5. **Feature flag** (alternativa a revert limpio): envolver el sampler en una policy `not_implemented` que matchea nada, dejándolo inerte sin re-deploy. Decisión: NO se implementa feature flag para mantener el cambio simple.

## Dependencies

- `otel/opentelemetry-collector-contrib:0.137.0` (ya pinneado en `docker-compose.yaml`). El `tail_sampling` processor está disponible en el build contrib desde 0.79.0+. No requiere bump de versión.
- `loadbalancing_exporter` y `tail_sampling` viven en packages separados; usar `tail_sampling` no requiere tocar `loadbalancing_exporter`. No relevante para este change.
- **No hay nuevas dependencias de aplicación** (no se agrega nada al `go.mod` del backend).

## Success Criteria

- [ ] `docker compose config` y `docker compose up -d` arrancan sin warnings nuevos en el log del otel-collector.
- [ ] `curl -sf http://localhost:13133/` (o el healthcheck del compose) sigue verde después del change.
- [ ] Generar 100 requests forzados: 90 exitosos, 10 con error 500. Confirmar en Jaeger UI que los 10 errores están presentes (100% retention de errores).
- [ ] Generar 1 request con latencia > 2s (puede ser un `time.Sleep(2500*time.Millisecond)` instrumentado en el backend). Confirmar en Jaeger que ese trace está presente (100% retention de lentos).
- [ ] Comparar el "total traces" en Jaeger antes/después con el mismo script de 1000 requests exitosos. La reducción debe ser ≥ 80% (objetivo: 95% ±).
- [ ] Memoria RSS del otel-collector en `docker stats` después de 5 minutos de tráfico sintético: < 400MB.
- [ ] Ningún cambio en el código Go del backend (`git diff backend/` muestra 0 líneas).
- [ ] PR diff total < 100 líneas (objetivo: < 50). Si se pasa, considerar chaining.

## Notes para la fase `sdd-spec`

Cuando `sdd-spec` arranque, tiene que producir **un único spec nuevo** en `openspec/changes/cachicamas-tail-sampling/specs/trace-tail-sampling/spec.md` con:

- Requirement 1: "El collector MUST evaluar cada trace completo antes de decidir exportación" (Given/When/Then).
- Requirement 2: "El collector MUST exportar el 100% de traces con error status o exception events" (Given/When/Then verificable con el test de los 10 errores).
- Requirement 3: "El collector MUST exportar el 100% de traces con latencia > 1s" (Given/When/Then verificable con el test del sleep instrumentado).
- Requirement 4: "El collector MUST muestrear probabilísticamente el tráfico exitoso a un rate configurable, default 5%" (Given/When/Then verificable con la métrica de volumen).
- Requirement 5: "El collector MUST tener una catch-all policy que asegure ≥ 1% de exportación incluso cuando ninguna otra policy matchea" (Given/When/Then).
- Requirement 6: "El memory_limiter MUST operar al 60% o menos del límite del container para dejar headroom al sampler" (Given/When/Then verificable con `docker stats`).

Reglas recordadas del `openspec/config.yaml`: Given/When/Then, RFC 2119 keywords, cada escenario independiente.

## Notes para la fase `sdd-design`

El design es chiquito — un solo archivo de config. `sdd-design` probablemente cabe en < 50 líneas: el diff del `collector-config.yaml` con rationale por bloque. Considerar merger design + tasks si el budget se siente ocioso. Llamada del executor: chain o no chain NO aplica (PR-único, scope acotado, < 100 líneas de diff esperado).

## Notes para la fase `sdd-tasks`

Tasks esperadas (preliminar, refinar en su fase):

- T1: Reordenar pipeline `traces` (insertar `tail_sampling` antes de `batch`).
- T2: Definir las 6 policies con comentarios inline en `collector-config.yaml`.
- T3: Ajustar `memory_limiter.limit_percentage` a 60.
- T4: Validar config con `docker compose up -d otel-collector` (debe arrancar sin errores).
- T5: Verificar que el healthcheck del compose sigue verde.
- T6: Generar tráfico sintético de prueba (script Go o curl) y comparar volumen Jaeger antes/después.
- T7: Verificar retention 100% de errores y lentos con tests forzados.
- T8: Actualizar `README.md` con la nota corta de 3-5 líneas.

Forecast del budget 400-line: **Low** (diff total esperado < 100 líneas, todo en un archivo, sin código Go). Single PR recomendado.
