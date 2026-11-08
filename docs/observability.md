# Observability (OpenObserve)

Self-hosted OpenObserve runs on the same machine as the Go API. The API exports request traces and logs over OTLP/HTTP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

## Start OpenObserve

`./scripts/run-local.sh` starts OpenObserve via Docker Compose and waits until `http://localhost:5080` responds before launching the API. Put `O2_PASS` (and optionally `O2_USER` / `O2_PORT`) in `.env.local`.

To skip OpenObserve: `SKIP_OPENOBSERVE=true ./scripts/run-local.sh`

Or start it alone:

```bash
docker compose up -d openobserve
```

UI: http://localhost:5080  
Login with `O2_USER` / `O2_PASS` (defaults in `.env.example`).

## Point the API at OpenObserve

Copy the OTEL block from `.env.example` into `.env.local`. Generate the Basic auth header:

```bash
printf '%s' "$O2_USER:$O2_PASS" | base64 -w0
# then set:
# OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic%20<that-value>
```

Default endpoint for same-machine Docker:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:5080/api/default
OTEL_SERVICE_NAME=zoo-api
```

If `OTEL_EXPORTER_OTLP_ENDPOINT` is unset, the API runs without OTLP export.

## Verify

```bash
./scripts/run-local.sh
curl -s http://localhost:8080/v1/healthz
```

In OpenObserve:

1. Open **Logs** and select the `default` stream (or the service stream).
2. Open **Traces** and look for spans named `zoo-api`.
3. Request log lines include `trace_id` / `span_id` when correlated with the active span.

## Notes

- No trailing slash on `OTEL_EXPORTER_OTLP_ENDPOINT` (exporters append `/v1/traces` and `/v1/logs`).
- Percent-encode the space in `Basic%20` inside `OTEL_EXPORTER_OTLP_HEADERS`.
- Data persists under `data/openobserve/` on this machine.
