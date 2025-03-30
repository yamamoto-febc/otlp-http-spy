# otlp-http-spy

`otlp-http-spy` is a lightweight HTTP proxy designed for inspecting OpenTelemetry Protocol (OTLP) traffic over HTTP.\
It captures and logs OTLP requests (logs, traces, metrics), forwards them to a configured backend, and displays both requests and responses in human-readable JSON.

## ✨ Features

- Supports OTLP over HTTP (`/v1/logs`, `/v1/traces`, `/v1/metrics`)
- Parses and logs OTLP Protobuf payloads as JSON
- Forwards requests to upstream endpoints
- Dumps HTTP request/response headers and bodies
- Configurable via environment variables
- Tiny Docker image (multi-arch)

> [!NOTE] 
> **Forwarding Behavior**: Requests are only forwarded if a target endpoint is configured using either `ENDPOINT` or one of the specific OTLP endpoint variables (`LOGS_ENDPOINT`, `TRACES_ENDPOINT`, `METRICS_ENDPOINT`).  
> If none of these are set, the proxy will simply log the incoming request without forwarding it.

---

## 🚀 Usage

You can run `otlp-http-spy` as a standalone binary or Docker container.

> [!WARNING] 
> **Supported Content-Type**: Only `application/x-protobuf` is supported.\
> Requests with other content types (e.g. `application/json`) will be rejected.

> [!NOTE] 
> **Compression Notice**: Gzip-compressed request bodies are currently **not supported**.\
> Please ensure your OTLP sender does not apply compression to outgoing requests.

### Run with Docker

```bash
docker run --rm -p 4318:4318 \
  -e ENDPOINT=http://otel-collector:4318 \
  ghcr.io/yamamoto-febc/otlp-http-spy:latest
```

### Run as binary

```bash
LISTEN_ADDR=:4318 \
ENDPOINT=http://otel-collector:4318 \
./otlp-http-spy
```

---

## ⚙️ Environment Variables

| Variable           | Description                            | Example                            |
| ------------------ | -------------------------------------- | ---------------------------------- |
| `LISTEN_ADDR`      | Address to listen on                   | `:4318`                            |
| `ENDPOINT`         | Common base URL for all OTLP endpoints | `http://localhost:4318`            |
| `LOGS_ENDPOINT`    | Custom endpoint for OTLP logs          | `http://localhost:4318/v1/logs`    |
| `TRACES_ENDPOINT`  | Custom endpoint for OTLP traces        | `http://localhost:4318/v1/traces`  |
| `METRICS_ENDPOINT` | Custom endpoint for OTLP metrics       | `http://localhost:4318/v1/metrics` |

If specific endpoints are not set, `ENDPOINT` is used as the base.

---

## 📤 Example Output

```text
===> Received OTLP request:  /v1/logs

=== HTTP Request Headers ===
POST /v1/logs HTTP/1.1
Host: localhost:4318
Content-Type: application/x-protobuf

=== OTLP Message (Request) ===

{
  "resourceLogs": [
    ...
  ]
}

=== Forwarded Response Headers ===
HTTP/1.1 200 OK
Content-Type: application/x-protobuf

=== OTLP Message (Response) ===

{
  "partialSuccess": {
    "rejectedLogRecords": 0
  }
}
```

---

## 🧪 Quick Example with `otel-cli`

You can test `otlp-http-spy` using [`otel-cli`](https://github.com/equinix-labs/otel-cli), a command-line tool for sending OTLP requests.

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
otel-cli exec --service my-service --name "curl google" curl https://google.com
```

This will send a trace span to `otlp-http-spy`, and you can inspect the received OTLP payload in the logs.

---

## 📦 Docker Image

- Image: `ghcr.io/yamamoto-febc/otlp-http-spy`
- Multi-platform: `linux/amd64`, `linux/arm64`
- Built with Go using CGO disabled (static binary)

---

## 📜 License

`otlp-http-spy` Copyright (C) 2025 Kazumichi Yamamoto (@yamamoto-febc)  
This project is published under [Apache 2.0 License](LICENSE).

