// Package drain wraps scontrol to drain failed nodes from the SLURM scheduler.
package drain

import (
    "context"
    "fmt"
    "log/slog"
    "os/exec"
    "path/filepath"
    "time"
)

// Drainer issues SLURM drain commands for unhealthy nodes.
type Drainer struct {
    binPath string
    reason  string
    wait    time.Duration
    log     *slog.Logger
}

func New(binPath, reason string, wait time.Duration, log *slog.Logger) *Drainer {
    return &Drainer{binPath: binPath, reason: reason, wait: wait, log: log}
}

// Drain marks a node as drained in SLURM and waits for running jobs to finish.
func (d *Drainer) Drain(ctx context.Context, hostname string) error {
    scontrol := filepath.Join(d.binPath, "scontrol")
    args := []string{"update", fmt.Sprintf("NodeName=%s", hostname),
        "State=DRAIN", fmt.Sprintf("Reason=%s", d.reason)}

    d.log.Info("draining node", "node", hostname, "reason", d.reason)
    cmd := exec.CommandContext(ctx, scontrol, args...)
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("scontrol drain %s: %w: %s", hostname, err, out)
    }

    // Wait for running jobs to vacate before returning
    deadline := time.Now().Add(d.wait)
    for time.Now().Before(deadline) {
        if !d.hasRunningJobs(ctx, hostname) {
            d.log.Info("node drained cleanly", "node", hostname)
            return nil
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(15 * time.Second):
        }
    }
    d.log.Warn("drain timeout: jobs still running", "node", hostname)
    return nil
}

func (d *Drainer) hasRunningJobs(ctx context.Context, hostname string) bool {
    squeue := filepath.Join(d.binPath, "squeue")
    cmd := exec.CommandContext(ctx, squeue, "-h", "-n", hostname, "-o", "%i")
    out, err := cmd.Output()
    return err == nil && len(out) > 0
}
