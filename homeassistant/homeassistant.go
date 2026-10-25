// https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery
package homeassistant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	publishTimeout = 10 * time.Second
	topicBase      = "akizuki-dg-route-b"
)

// Home Assistant のセンサー
type Sensor struct {
	Key               string
	Name              string
	Platform          string
	DeviceClass       string
	StateClass        string
	Unit              string
	EntityCategory    string
	SuggestedDecimals int
}

var Sensors = []Sensor{
	{Key: "power", Name: "瞬時電力", Platform: "sensor", DeviceClass: "power", StateClass: "measurement", Unit: "W"},
	{Key: "current_r", Name: "瞬時電流 R相", Platform: "sensor", DeviceClass: "current", StateClass: "measurement", Unit: "A", SuggestedDecimals: 1},
	{Key: "current_t", Name: "瞬時電流 T相", Platform: "sensor", DeviceClass: "current", StateClass: "measurement", Unit: "A", SuggestedDecimals: 1},
	{Key: "energy_normal", Name: "積算電力量 正方向", Platform: "sensor", DeviceClass: "energy", StateClass: "total_increasing", Unit: "kWh"},
	{Key: "energy_reverse", Name: "積算電力量 逆方向", Platform: "sensor", DeviceClass: "energy", StateClass: "total_increasing", Unit: "kWh"},
	{Key: "transmission_time", Name: "直近1時間の送信時間", Platform: "sensor", DeviceClass: "duration", StateClass: "measurement", Unit: "s", EntityCategory: "diagnostic", SuggestedDecimals: 1},
	{Key: "transmission_rate_limited", Name: "送信時間制限中", Platform: "binary_sensor", DeviceClass: "problem", EntityCategory: "diagnostic"},
}

type Config struct {
	Broker          string
	Username        string
	Password        string
	DiscoveryPrefix string
	NodeID          string
	Version         string
	Logger          *slog.Logger
}

type Client struct {
	client          mqtt.Client
	discoveryPrefix string
	logger          *slog.Logger
	nodeID          string
	version         string

	mu                 sync.Mutex
	available          bool
	currentPhasesKnown bool
	currentR           bool
	currentT           bool
}

func New(c Config) *Client {
	h := &Client{
		discoveryPrefix: c.DiscoveryPrefix,
		logger:          c.Logger,
		nodeID:          strings.ReplaceAll(c.NodeID, ":", ""),
		version:         c.Version,
	}

	opts := mqtt.NewClientOptions().
		AddBroker(c.Broker).
		SetClientID(topicBase+"-"+h.nodeID).
		SetUsername(c.Username).
		SetPassword(c.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetWill(h.topic("availability"), "offline", 1, true).
		SetOnConnectHandler(func(mqtt.Client) {
			h.logger.Info("Connected to MQTT broker")
			// 再接続時は LWT で offline になっているので設定と状態を送り直す
			if err := h.publishDiscovery(); err != nil {
				h.logger.Error("Failed to build discovery payload", "err", err)
			}
			h.mu.Lock()
			available := h.available
			h.mu.Unlock()
			h.publishAvailability(available)
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			h.logger.Warn("Lost connection to MQTT broker", "err", err)
		})

	h.client = mqtt.NewClient(opts)
	// SetConnectRetry によりブローカーに繋がるまでバックグラウンドで再試行される
	h.client.Connect()

	return h
}

func (h *Client) topic(key string) string {
	return fmt.Sprintf("%s/%s/%s", topicBase, h.nodeID, key)
}

// ブローカーが落ちていても計測ループを止めないよう、完了は待たずに結果だけログに出す
func (h *Client) publish(topic string, retained bool, payload any) mqtt.Token {
	t := h.client.Publish(topic, 1, retained, payload)
	go func() {
		if !t.WaitTimeout(publishTimeout) {
			h.logger.Warn("Publish timed out", "topic", topic)
			return
		}
		if err := t.Error(); err != nil {
			h.logger.Error("Failed to publish", "topic", topic, "err", err)
		}
	}()
	return t
}

// 初回応答までは、非対応の相を一時的に登録しないよう Discovery を待つ。
func (h *Client) publishDiscovery() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.currentPhasesKnown {
		return nil
	}
	components := map[string]map[string]any{}
	for _, s := range Sensors {
		if (s.Key == "current_r" && !h.currentR) || (s.Key == "current_t" && !h.currentT) {
			continue
		}
		c := map[string]any{
			"p":           s.Platform,
			"name":        s.Name,
			"unique_id":   h.nodeID + "_" + s.Key,
			"state_topic": h.topic(s.Key),
		}
		if s.DeviceClass != "" {
			c["device_class"] = s.DeviceClass
		}
		if s.StateClass != "" {
			c["state_class"] = s.StateClass
		}
		if s.Unit != "" {
			c["unit_of_measurement"] = s.Unit
		}
		if s.EntityCategory != "" {
			c["entity_category"] = s.EntityCategory
		}
		if s.SuggestedDecimals != 0 {
			c["suggested_display_precision"] = s.SuggestedDecimals
		}
		components[s.Key] = c
	}

	payload, err := json.Marshal(map[string]any{
		"device": map[string]any{
			"identifiers": []string{h.nodeID},
			"name":        "スマートメーター",
			"model":       "B ルート (MB-RL7023-11)",
		},
		"origin": map[string]any{
			"name":       "akizuki-dg-route-b-exporter",
			"sw_version": h.version,
			"url":        "https://github.com/rokoucha/akizuki-dg-route-b-exporter",
		},
		"availability_topic": h.topic("availability"),
		"components":         components,
	})
	if err != nil {
		return err
	}

	h.publish(fmt.Sprintf("%s/device/%s/config", h.discoveryPrefix, h.nodeID), true, payload)
	return nil
}

// 電流プロパティの応答から、計測できる相だけを Discovery に含める。
func (h *Client) SetCurrentPhases(r, t bool) {
	h.mu.Lock()
	changed := !h.currentPhasesKnown || h.currentR != r || h.currentT != t
	h.currentPhasesKnown, h.currentR, h.currentT = true, r, t
	h.mu.Unlock()
	if changed {
		if err := h.publishDiscovery(); err != nil {
			h.logger.Error("Failed to build discovery payload", "err", err)
		}
	}
}

func (h *Client) publishAvailability(available bool) mqtt.Token {
	payload := "offline"
	if available {
		payload = "online"
	}
	return h.publish(h.topic("availability"), true, payload)
}

// スマートメーターと通信できているかを通知します。
func (h *Client) SetAvailable(available bool) {
	h.mu.Lock()
	h.available = available
	h.mu.Unlock()
	h.publishAvailability(available)
}

// センサーの値を通知します。
func (h *Client) Publish(key string, value any) {
	var payload string
	switch v := value.(type) {
	case bool:
		payload = "OFF"
		if v {
			payload = "ON"
		}
	default:
		payload = fmt.Sprint(v)
	}
	h.publish(h.topic(key), false, payload)
}

func (h *Client) Close() {
	h.mu.Lock()
	h.available = false
	h.mu.Unlock()
	h.publishAvailability(false).WaitTimeout(publishTimeout)
	h.client.Disconnect(250)
}
