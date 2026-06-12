package main

import (
    "context"
    "flag"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/shravyavorugallu/node-watchdog/internal/checker"
    "github.com/shravyavorugallu/node-watchdog/internal/config"
    "github.com/shravyavorugallu/node-watchdog/internal/drain"
    "github.com/shravyavorugallu/node-watchdog/internal/notifier"
)

func main() {
    cfgPath := flag.String("config", "config.yaml", "path to config file")
    flag.Parse()

    log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

    cfg, err := config.Load(*cfgPath)
    if err != nil {
        log.Error("failed to load config", "err", err)
        os.Exit(1)
    }

    chk     := checker.New(cfg.Cluster.Name, cfg.Check.HTTPPort, cfg.Check.Timeout)
    drainer := drain.New(cfg.Drain.SlurmBinPath, cfg.Drain.DrainReason, cfg.Drain.WaitDrain, log)
    ntfr    := notifier.New(cfg.Notify.SlackWebhook, cfg.Notify.Channel)

    // Expose Prometheus metrics
    mux := http.NewServeMux()
    mux.Handle(cfg.Metrics.Path, promhttp.Handler())
    srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Metrics.Port), Handler: mux}
    go func() {
        log.Info("metrics server started", "port", cfg.Metrics.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Error("metrics server error", "err", err)
        }
    }()

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    ticker := time.NewTicker(cfg.Check.Interval)
    defer ticker.Stop()

    log.Info("watchdog started", "cluster", cfg.Cluster.Name, "nodes", len(cfg.Cluster.Nodes))
    for {
        select {
        case <-ctx.Done():
            log.Info("shutting down")
            return
        case <-ticker.C:
            failed := chk.CheckAll(ctx, cfg.Cluster.Nodes, cfg.Check.FailCount)
            for _, host := range failed {
                log.Warn("node failed health checks", "node", host)
                if cfg.Drain.Enabled {
                    if err := drainer.Drain(ctx, host); err != nil {
                        log.Error("drain failed", "node", host, "err", err)
                    }
                }
                if err := ntfr.NodeFailed(cfg.Cluster.Name, host, cfg.Check.FailCount); err != nil {
                    log.Warn("slack notification failed", "err", err)
                }
            }
        }
    }
}
