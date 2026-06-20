# Jaeger

This directory is reserved for future Jaeger-specific configuration
(e.g. sampling strategies, persistent storage backends, UI auth).

Today Jaeger v2 runs in **all-in-one** mode from the official
`jaegertracing/jaeger:2.0.0` image — no config file needed. Traces are
received via its built-in OTLP receiver (gRPC `:4317`, HTTP `:4318`) and
the UI is served on `:16686`.

When you outgrow the in-memory store, drop a `config.yaml` here pointing
at an external storage backend (Elasticsearch, Cassandra, Kafka) and
mount it in `docker-compose.yaml`.