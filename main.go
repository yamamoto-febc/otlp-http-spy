package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/caarlos0/env/v11"
	protoLogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	protoMetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	protoTrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var listenAddr = ":8080"

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}
	cfg.Init()
	logConfiguredEndpoints(cfg)

	http.HandleFunc("/v1/logs", handleLogs)
	http.HandleFunc("/v1/traces", handleTraces)
	http.HandleFunc("/v1/metrics", handleMetrics)
	log.Println("Starting OTLP/HTTP spy on ", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, nil))
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, &protoLogs.ExportLogsServiceRequest{})
}

func handleTraces(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, &protoTrace.ExportTraceServiceRequest{})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, &protoMetrics.ExportMetricsServiceRequest{})
}

func handleRequest(w http.ResponseWriter, r *http.Request, message proto.Message) {
	buf := &bytes.Buffer{}

	logRequestReceived(buf, r.URL.Path)
	logHTTPRequest(buf, r)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := proto.Unmarshal(body, message); err != nil {
		log.Printf("Failed to parse OTLP logs: %v", err)
		http.Error(w, "invalid protobuf", http.StatusBadRequest)
		return
	}

	logProtoMessage(buf, message)

	log.Println(buf.String())
	w.WriteHeader(http.StatusOK)
}

func logConfiguredEndpoints(cfg Config) {
	if cfg.LogsEndpoint != "" {
		log.Println("[Proxy] Logs will be forwarded to    =>", cfg.LogsEndpoint)
	}
	if cfg.TracesEndpoint != "" {
		log.Println("[Proxy] Traces will be forwarded to  =>", cfg.TracesEndpoint)
	}
	if cfg.MetricsEndpoint != "" {
		log.Println("[Proxy] Metrics will be forwarded to =>", cfg.MetricsEndpoint)
	}
}

func logRequestReceived(w io.Writer, path string) {
	fmt.Fprintln(w, "===> Received OTLP request: ", path)
	fmt.Fprintln(w, "")
}

func logHTTPRequest(w io.Writer, req *http.Request) {
	dump, err := httputil.DumpRequest(req, false)
	if err != nil {
		log.Println("Failed to dump request: ", err)
		return
	}

	fmt.Fprintln(w, "=== HTTP Request Headers ===")
	fmt.Fprintln(w, string(dump))
}

func logProtoMessage(w io.Writer, m proto.Message) {
	message, err := marshalProtoMessage(m)
	if err != nil {
		log.Printf("Failed to marshal to JSON: %v", err)
	}
	fmt.Fprintln(w, "=== OTLP Message ===")
	fmt.Fprintln(w, string(message))
}

func marshalProtoMessage(m proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(m)
}
