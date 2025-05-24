package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaBroker     = getEnv("KAFKA_BROKER", "localhost:9092")
	impressionTopic = getEnv("IMPRESSION_TOPIC", "ad-impressions")
	clickTopic      = getEnv("CLICK_TOPIC", "ad-clicks")
	eventRate       = getEnvInt("EVENT_RATE", 50)
	clickRatio      = getEnvFloat("CLICK_RATIO", 0.1)

	campaigns = generateStringSlice("camp-", 10)
	ads       = generateStringSlice("ad-", 100)
	devices   = []string{"mobile", "desktop", "tablet"}
	browsers  = []string{"chrome", "safari", "firefox", "edge"}
	userPool  = 10000

	campaignClickBoost = make(map[string]float64)
	mu                 sync.RWMutex
)

type Impression struct {
	ImpressionID string  `json:"impression_id"`
	UserID       string  `json:"user_id"`
	CampaignID   string  `json:"campaign_id"`
	AdID         string  `json:"ad_id"`
	DeviceType   string  `json:"device_type"`
	Browser      string  `json:"browser"`
	Timestamp    int64   `json:"event_timestamp"`
	Cost         float64 `json:"cost"`
}

type Click struct {
	ClickID      string `json:"click_id"`
	ImpressionID string `json:"impression_id"`
	UserID       string `json:"user_id"`
	Timestamp    int64  `json:"event_timestamp"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("Waiting for Kafka to be ready...")
	time.Sleep(10 * time.Second)

	initCampaignBoost()

	writerImpression := newKafkaWriter(impressionTopic)
	writerClick := newKafkaWriter(clickTopic)

	defer writerImpression.Close()
	defer writerClick.Close()

	ticker := time.NewTicker(time.Second / time.Duration(eventRate))
	behaviorTicker := time.NewTicker(10 * time.Minute)

	fmt.Println("Starting event generation...")

	for {
		select {
		case <-ticker.C:
			imp := generateImpression()
			impBytes, _ := json.Marshal(imp)
			_ = writerImpression.WriteMessages(context.Background(),
				kafka.Message{
					Key:   []byte(imp.ImpressionID),
					Value: impBytes,
				})

			mu.RLock()
			actualRatio := clickRatio * campaignClickBoost[imp.CampaignID]
			mu.RUnlock()

			if rand.Float64() < actualRatio {
				delay := rand.Intn(9500) + 500
				time.Sleep(time.Duration(delay) * time.Millisecond)

				clk := Click{
					ClickID:      uuid.New().String(),
					ImpressionID: imp.ImpressionID,
					UserID:       imp.UserID,
					Timestamp:    imp.Timestamp + int64(delay),
				}

				clkBytes, _ := json.Marshal(clk)
				_ = writerClick.WriteMessages(context.Background(),
					kafka.Message{
						Key:   []byte(clk.ClickID),
						Value: clkBytes,
					})
			}

		case <-behaviorTicker.C:
			fmt.Println("Updating campaign performance...")
			updateCampaignBoost()
		}
	}
}

func generateImpression() Impression {
	return Impression{
		ImpressionID: uuid.New().String(),
		UserID:       fmt.Sprintf("user-%d", rand.Intn(userPool)+1),
		CampaignID:   campaigns[rand.Intn(len(campaigns))],
		AdID:         ads[rand.Intn(len(ads))],
		DeviceType:   devices[rand.Intn(len(devices))],
		Browser:      browsers[rand.Intn(len(browsers))],
		Timestamp:    time.Now().UnixNano() / 1e6,
		Cost:         round(rand.Float64()*0.49 + 0.01),
	}
}

func newKafkaWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(kafkaBroker, ",")...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
}

func generateStringSlice(prefix string, count int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = fmt.Sprintf("%s%d", prefix, i+1)
	}
	return result
}

func round(val float64) float64 {
	return float64(int(val*100)) / 100
}

func initCampaignBoost() {
	for _, camp := range campaigns {
		campaignClickBoost[camp] = rand.Float64()*0.7 + 0.8
	}
}

func updateCampaignBoost() {
	mu.Lock()
	defer mu.Unlock()

	for _, camp := range campaigns {
		campaignClickBoost[camp] = rand.Float64()*0.7 + 0.8
	}

	anomaly := campaigns[rand.Intn(len(campaigns))]
	if rand.Float64() < 0.5 {
		campaignClickBoost[anomaly] = rand.Float64()*1.0 + 2.0
		fmt.Printf("Anomaly: High CTR for %s\n", anomaly)
	} else {
		campaignClickBoost[anomaly] = rand.Float64()*0.2 + 0.1
		fmt.Printf("Anomaly: Low CTR for %s\n", anomaly)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
