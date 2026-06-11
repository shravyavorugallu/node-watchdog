// Package notifier sends Slack messages for node failures and recoveries.
package notifier

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Notifier struct {
    webhook string
    channel string
    client  *http.Client
}

func New(webhook, channel string) *Notifier {
    return &Notifier{
        webhook: webhook,
        channel: channel,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (n *Notifier) NodeFailed(cluster, host string, fails int) error {
    text := fmt.Sprintf(":red_circle: *[%s]* Node `%s` failed %d consecutive health checks — draining from SLURM",
        cluster, host, fails)
    return n.send(text)
}

func (n *Notifier) NodeRecovered(cluster, host string) error {
    text := fmt.Sprintf(":large_green_circle: *[%s]* Node `%s` is healthy again",
        cluster, host)
    return n.send(text)
}

func (n *Notifier) send(text string) error {
    payload, _ := json.Marshal(map[string]string{
        "channel": n.channel,
        "text":    text,
    })
    resp, err := n.client.Post(n.webhook, "application/json", bytes.NewReader(payload))
    if err != nil {
        return err
    }
    resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack returned %d", resp.StatusCode)
    }
    return nil
}
