package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the observability platform.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	GRPC       GRPCConfig       `yaml:"grpc"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Alerting   AlertingConfig   `yaml:"alerting"`
	Retention  RetentionConfig  `yaml:"retention"`
	Tenancy    TenancyConfig    `yaml:"tenancy"`
}

type ServerConfig struct {
	HTTPAddr string `yaml:"http_addr"`
	GRPCAddr string `yaml:"grpc_addr"`
}

type GRPCConfig struct {
	MaxRecvMsgSize int `yaml:"max_recv_msg_size"`
	MaxSendMsgSize int `yaml:"max_send_msg_size"`
}

type ClickHouseConfig struct {
	Addrs    []string `yaml:"addrs"`
	Database string   `yaml:"database"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type PrometheusConfig struct {
	ScrapeInterval time.Duration `yaml:"scrape_interval"`
	ListenAddr     string        `yaml:"listen_addr"`
}

type AlertingConfig struct {
	EvaluationInterval time.Duration       `yaml:"evaluation_interval"`
	Channels           []NotificationChannel `yaml:"channels"`
}

type NotificationChannel struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"` // webhook, slack, email
	Config  map[string]string `yaml:"config"`
}

type RetentionConfig struct {
	LogsTTL    time.Duration `yaml:"logs_ttl"`
	TracesTTL  time.Duration `yaml:"traces_ttl"`
	MetricsTTL time.Duration `yaml:"metrics_ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

type TenancyConfig struct {
	Enabled        bool   `yaml:"enabled"`
	HeaderName     string `yaml:"header_name"`
	DefaultTenant  string `yaml:"default_tenant"`
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPAddr: ":8080",
			GRPCAddr: ":9090",
		},
		GRPC: GRPCConfig{
			MaxRecvMsgSize: 16 << 20, // 16MB
			MaxSendMsgSize: 16 << 20,
		},
		ClickHouse: ClickHouseConfig{
			Addrs:    []string{"localhost:9000"},
			Database: "observability",
			Username: "default",
			Password: "",
		},
		Prometheus: PrometheusConfig{
			ScrapeInterval: 15 * time.Second,
			ListenAddr:     ":9091",
		},
		Alerting: AlertingConfig{
			EvaluationInterval: 30 * time.Second,
		},
		Retention: RetentionConfig{
			LogsTTL:         30 * 24 * time.Hour,
			TracesTTL:       7 * 24 * time.Hour,
			MetricsTTL:      90 * 24 * time.Hour,
			CleanupInterval: 1 * time.Hour,
		},
		Tenancy: TenancyConfig{
			Enabled:       false,
			HeaderName:    "X-Tenant-ID",
			DefaultTenant: "default",
		},
	}
}
