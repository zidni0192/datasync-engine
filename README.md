# datasync-engine

A local-first event ingestion prototype. The first slice accepts events over HTTP, assigns monotonically increasing in-memory IDs, and reads them back using an ID cursor. The in-memory store is intentionally replaceable; no external service or Docker is needed.

## Requirements

- Go 1.26.2 or newer

## Run

```sh
go run .
```

The server listens on `:8080`.

```sh
curl -i http://localhost:8080/healthz
curl -i -X POST http://localhost:8080/events \
  -H 'Content-Type: application/json' \
  -d '{"topic":"orders","payload":"created"}'
curl 'http://localhost:8080/events?after=0&limit=100'
```

`POST /events` requires non-empty `topic` and `payload` strings and returns the event with its assigned ID. `GET /events` supports `after` (exclusive, default `0`) and `limit` (1–500, default `100`). `GET /healthz` returns 204. Data is held in memory and disappears when the process exits; this is a demo, not durable storage.

## Checks

```sh
go test ./...
go test -race ./...
go vet ./...
bash scripts/check.sh
```

## Architecture

`net/http` routes requests to a handler backed by `EventStore`. The store owns ID allocation and synchronization; handlers validate and encode the API contract. Keeping storage behind a small in-process type allows later persistence and integration tests without changing the HTTP surface. Tests use `httptest` and synthetic payloads only.
