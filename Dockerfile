# builder
FROM golang:1.24 AS builder
LABEL maintainer="Kazumichi Yamamoto <yamamoto.febc@gmail.com>"

WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
RUN go build -o otlp-http-spy -ldflags="-s -w"

# runtime
FROM alpine:3.21
LABEL maintainer="Kazumichi Yamamoto <yamamoto.febc@gmail.com>"

RUN apk add --no-cache ca-certificates \
  && adduser -D -u 10001 otlpuser

COPY --from=builder /app/otlp-http-spy /usr/bin/

USER otlpuser
ENTRYPOINT ["/usr/bin/otlp-http-spy"]
