# node-watchdog

Go service that watches HPC cluster nodes via HTTP health checks, detects failures, auto-drains them from SLURM, and sends Slack notifications. Ships as a single static binary with a systemd unit.

## How it works

```
every 30s:
  probe all nodes via node_exporter HTTP endpoint (concurrent)
    |
  3 consecutive failures?
    yes: scontrol drain <node> + Slack alert
    no:  update Prometheus gauge, continue
```

## Metrics exposed

| Metric | Description |
|---|---|
| `watchdog_node_healthy` | 1 if node passed last check |
| `watchdog_node_consecutive_failures` | Current failure streak |
| `watchdog_probe_total` | Total probes per node |
| `watchdog_probe_failures_total` | Total failures per node |

## Build and run

```bash
go build -o bin/node-watchdog ./cmd/watchdog
export SLACK_WEBHOOK_URL=https://hooks.slack.com/...
./bin/node-watchdog --config config.yaml
```

Or install as a systemd service:

```bash
sudo cp bin/node-watchdog /usr/local/bin/
sudo cp systemd/node-watchdog.service /etc/systemd/system/
sudo systemctl enable --now node-watchdog
```

## Test

```bash
go test ./... -v -race
```

## Tech

Go 1.22 | prometheus/client_golang | slog | scontrol (SLURM) | systemd
