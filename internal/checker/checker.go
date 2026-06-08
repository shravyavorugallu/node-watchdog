// Package checker probes cluster nodes via HTTP (node_exporter) and ICMP.
package checker

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    probeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "watchdog_probe_total",
        Help: "Total node probe attempts",
    }, []string{"cluster", "node"})

    probeFailures = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "watchdog_probe_failures_total",
        Help: "Total failed node probes",
    }, []string{"cluster", "node"})

    nodeHealthy = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "watchdog_node_healthy",
        Help: "1 if node passed last health check",
    }, []string{"cluster", "node"})

    consecutiveFails = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "watchdog_node_consecutive_failures",
        Help: "Consecutive health check failures",
    }, []string{"cluster", "node"})
)

// NodeState tracks per-node failure count.
type NodeState struct {
    Hostname string
    Fails    int
    Healthy  bool
}

// Checker runs parallel health checks against all nodes.
type Checker struct {
    cluster  string
    httpPort int
    timeout  time.Duration
    client   *http.Client
    states   map[string]*NodeState
    mu       sync.Mutex
}

func New(cluster string, httpPort int, timeout time.Duration) *Checker {
    return &Checker{
        cluster:  cluster,
        httpPort: httpPort,
        timeout:  timeout,
        client:   &http.Client{Timeout: timeout},
        states:   make(map[string]*NodeState),
    }
}

// CheckAll probes every node concurrently and returns newly failed nodes.
func (c *Checker) CheckAll(ctx context.Context, nodes []string, failThreshold int) []string {
    type result struct {
        host   string
        ok     bool
    }
    ch := make(chan result, len(nodes))

    for _, node := range nodes {
        go func(h string) {
            ch <- result{host: h, ok: c.probe(ctx, h)}
        }(node)
    }

    var newFails []string
    for range nodes {
        r := <-ch
        c.mu.Lock()
        s, exists := c.states[r.host]
        if !exists {
            s = &NodeState{Hostname: r.host, Healthy: true}
            c.states[r.host] = s
        }

        probeTotal.WithLabelValues(c.cluster, r.host).Inc()
        if r.ok {
            s.Fails = 0
            s.Healthy = true
            nodeHealthy.WithLabelValues(c.cluster, r.host).Set(1)
        } else {
            s.Fails++
            probeFailures.WithLabelValues(c.cluster, r.host).Inc()
            nodeHealthy.WithLabelValues(c.cluster, r.host).Set(0)
            if s.Fails == failThreshold {
                newFails = append(newFails, r.host)
                s.Healthy = false
            }
        }
        consecutiveFails.WithLabelValues(c.cluster, r.host).Set(float64(s.Fails))
        c.mu.Unlock()
    }
    return newFails
}

// probe sends a GET to the node's node_exporter /metrics endpoint.
func (c *Checker) probe(ctx context.Context, host string) bool {
    url := fmt.Sprintf("http://%s:%d/metrics", host, c.httpPort)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return false
    }
    resp, err := c.client.Do(req)
    if err != nil {
        return false
    }
    resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}
