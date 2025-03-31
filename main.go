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

var config Config

type Config struct {
	ListenAddr      string `env:"LISTEN_ADDR" envDefault:":4318"`
	Endpoint        string `env:"ENDPOINT"`
	LogsEndpoint    string `env:"LOGS_ENDPOINT"`
	TracesEndpoint  string `env:"TRACES_ENDPOINT"`
	MetricsEndpoint string `env:"METRICS_ENDPOINT"`
}

func (c *Config) Init() {
	if c.LogsEndpoint == "" && c.Endpoint != "" {
		c.LogsEndpoint = c.Endpoint + "/v1/logs"
	}
	if c.TracesEndpoint == "" && c.Endpoint != "" {
		c.TracesEndpoint = c.Endpoint + "/v1/traces"
	}
	if c.MetricsEndpoint == "" && c.Endpoint != "" {
		c.MetricsEndpoint = c.Endpoint + "/v1/metrics"
	}
}

type protoRequestResponse struct {
	request  proto.Message
	response proto.Message
}

func getProtoRequestResponse(tp string) protoRequestResponse {
	switch tp {
	case "logs":
		return protoRequestResponse{
			request:  &protoLogs.ExportLogsServiceRequest{},
			response: &protoLogs.ExportLogsServiceResponse{},
		}
	case "traces":
		return protoRequestResponse{
			request:  &protoTrace.ExportTraceServiceRequest{},
			response: &protoTrace.ExportTraceServiceResponse{},
		}
	case "metrics":
		return protoRequestResponse{
			request:  &protoMetrics.ExportMetricsServiceRequest{},
			response: &protoMetrics.ExportMetricsServiceResponse{},
		}
	}
	panic("invalid type")
}

func main() {
	if err := env.Parse(&config); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}
	config.Init()
	logConfiguredEndpoints(config)

	http.HandleFunc("/v1/logs", handleLogs)
	http.HandleFunc("/v1/traces", handleTraces)
	http.HandleFunc("/v1/metrics", handleMetrics)
	log.Println("Starting OTLP/HTTP spy on ", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, getProtoRequestResponse("logs"), config.LogsEndpoint)
}

func handleTraces(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, getProtoRequestResponse("traces"), config.TracesEndpoint)
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, getProtoRequestResponse("metrics"), config.MetricsEndpoint)
}

func handleRequest(w http.ResponseWriter, r *http.Request, protoMessage protoRequestResponse, forwardTo string) {
	buf := &bytes.Buffer{}

	logRequestReceived(buf, r.URL.Path)
	logHTTPRequest(buf, r)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := proto.Unmarshal(body, protoMessage.request); err != nil {
		log.Printf("Failed to parse OTLP logs: %v", err)
		http.Error(w, "invalid protobuf", http.StatusBadRequest)
		return
	}

	logProtoMessage(buf, protoMessage.request, "Request")

	if forwardTo != "" {
		resp, err := forwardRequest(buf, forwardTo, body, r, protoMessage.response)
		if err != nil {
			log.Printf("Forwarding failed: %v", err)
			http.Error(w, "failed to forward request", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			http.Error(w, "failed to read response", http.StatusInternalServerError)
			return
		}

		if err := proto.Unmarshal(respBytes, protoMessage.response); err != nil {
			logRawResponseBody(buf, respBytes)
		} else {
			logProtoMessage(buf, protoMessage.response, "Response")
		}

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		if _, err := w.Write(respBytes); err != nil {
			log.Printf("Failed to write response body to client: %v", err)
		}
	}

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

	fmt.Fprint(w, "=== HTTP Request Headers ===\n\n")
	fmt.Fprintln(w, string(dump))
}

func logHTTPResponse(w io.Writer, resp *http.Response) {
	fmt.Fprint(w, "=== Forwarded Response Headers ===\n\n")
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		log.Println("Failed to dump response: ", err)
		return
	}
	fmt.Fprintln(w, string(dump))
}

func logRawResponseBody(w io.Writer, respData []byte) {
	fmt.Fprint(w, "=== Raw Response ===\n\n")
	fmt.Fprintln(w, string(respData))
	fmt.Fprintln(w, "")
}

func logProtoMessage(w io.Writer, m proto.Message, t string) {
	message, err := marshalProtoMessage(m)
	if err != nil {
		log.Printf("Failed to marshal to JSON: %v", err)
	}
	fmt.Fprintf(w, "=== OTLP Message (%s) ===\n\n", t)
	fmt.Fprintln(w, string(message))
	fmt.Fprintln(w, "")
}

func marshalProtoMessage(m proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(m)
}

func forwardRequest(buf io.Writer, endpoint string, body []byte, original *http.Request, responseMessage proto.Message) (*http.Response, error) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create forward request: %w", err)
	}

	for key, values := range original.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	logHTTPResponse(buf, resp)
	return resp, nil
}
