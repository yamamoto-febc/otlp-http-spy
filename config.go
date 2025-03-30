package main

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
