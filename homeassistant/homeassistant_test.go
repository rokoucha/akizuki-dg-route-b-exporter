package homeassistant

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type completedToken struct{}

func (completedToken) Wait() bool                     { return true }
func (completedToken) WaitTimeout(time.Duration) bool { return true }
func (completedToken) Done() <-chan struct{}          { ch := make(chan struct{}); close(ch); return ch }
func (completedToken) Error() error                   { return nil }

type discoveryClient struct {
	mqtt.Client
	payloads [][]byte
}

func (c *discoveryClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	if topic != "homeassistant/device/test/config" || qos != 1 || !retained {
		panic("unexpected discovery publication")
	}
	c.payloads = append(c.payloads, payload.([]byte))
	return completedToken{}
}

func TestCurrentPhaseDiscovery(t *testing.T) {
	c := &discoveryClient{}
	h := &Client{client: c, discoveryPrefix: "homeassistant", nodeID: "test", logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := h.publishDiscovery(); err != nil {
		t.Fatal(err)
	}
	if len(c.payloads) != 0 {
		t.Fatal("discovered phases before first meter response")
	}
	for _, tc := range []struct {
		name  string
		r, t  bool
		count int
	}{
		{"single phase", true, false, 1},
		{"unchanged", true, false, 1},
		{"two phases", true, true, 2},
		{"unsupported currents", false, false, 3},
		{"T only", false, true, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h.SetCurrentPhases(tc.r, tc.t)
			if len(c.payloads) != tc.count {
				t.Fatalf("publications = %d, want %d", len(c.payloads), tc.count)
			}
			var payload struct {
				Components map[string]json.RawMessage `json:"components"`
			}
			if err := json.Unmarshal(c.payloads[len(c.payloads)-1], &payload); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]bool{"current_r": tc.r, "current_t": tc.t, "power": true, "energy_normal": true, "transmission_time": true} {
				if _, ok := payload.Components[key]; ok != want {
					t.Errorf("component %s present = %v, want %v", key, ok, want)
				}
			}
			// 再接続で送る定義も、判定済みの対応相を維持する。
			if err := h.publishDiscovery(); err != nil {
				t.Fatal(err)
			}
			if string(c.payloads[len(c.payloads)-1]) != string(c.payloads[len(c.payloads)-2]) {
				t.Fatal("reconnect changed discovery")
			}
			c.payloads = c.payloads[:len(c.payloads)-1]
		})
	}
}
