package config

import (
    "os"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Cluster   ClusterConfig   `yaml:"cluster"`
    Check     CheckConfig     `yaml:"check"`
    Drain     DrainConfig     `yaml:"drain"`
    Notify    NotifyConfig    `yaml:"notify"`
    Metrics   MetricsConfig   `yaml:"metrics"`
}

type ClusterConfig struct {
    Name  string   `yaml:"name"`
    Nodes []string `yaml:"nodes"`
}

type CheckConfig struct {
    Interval    time.Duration `yaml:"interval"`
    Timeout     time.Duration `yaml:"timeout"`
    FailCount   int           `yaml:"fail_count"`   // consecutive failures before action
    HTTPPort    int           `yaml:"http_port"`    // node_exporter port
}

type DrainConfig struct {
    Enabled       bool          `yaml:"enabled"`
    SlurmBinPath  string        `yaml:"slurm_bin_path"`
    DrainReason   string        `yaml:"drain_reason"`
    WaitDrain     time.Duration `yaml:"wait_drain"`
}

type NotifyConfig struct {
    SlackWebhook string `yaml:"slack_webhook"`
    Channel      string `yaml:"channel"`
}

type MetricsConfig struct {
    Port int    `yaml:"port"`
    Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(b, &cfg); err != nil {
        return nil, err
    }
    // defaults
    if cfg.Check.Interval == 0    { cfg.Check.Interval  = 30 * time.Second }
    if cfg.Check.Timeout == 0     { cfg.Check.Timeout   = 5  * time.Second }
    if cfg.Check.FailCount == 0   { cfg.Check.FailCount  = 3 }
    if cfg.Check.HTTPPort == 0    { cfg.Check.HTTPPort   = 9100 }
    if cfg.Metrics.Port == 0      { cfg.Metrics.Port     = 9201 }
    if cfg.Metrics.Path == ""     { cfg.Metrics.Path     = "/metrics" }
    if cfg.Drain.SlurmBinPath == "" { cfg.Drain.SlurmBinPath = "/usr/bin" }
    if cfg.Drain.DrainReason == "" { cfg.Drain.DrainReason = "node-watchdog: health check failed" }
    if cfg.Drain.WaitDrain == 0   { cfg.Drain.WaitDrain = 2 * time.Minute }
    return &cfg, nil
}
