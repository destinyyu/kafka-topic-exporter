package main

import (
	"context"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type scrapeConfig struct {
	Topics     []string
	DropLabels []string
}

func newScrapeConfig(topics, dropLabels string) scrapeConfig {
	return scrapeConfig{
		Topics:     strings.Split(topics, ","),
		DropLabels: strings.Split(dropLabels, ","),
	}
}

func startKafkaScraper(wg *sync.WaitGroup, shutdown chan struct{}, kafka sarama.ConsumerGroup, cfg scrapeConfig) {
	setDropLabels(cfg.DropLabels)
	go func() {
		wg.Add(1)
		defer wg.Done()

		ctx := context.Background()

		go func() {
			for {
				handler := ConsumerHandler{}
				if err := kafka.Consume(ctx, cfg.Topics, handler); err != nil {
					panic(err)
				}
			}
		}()

		<-shutdown
		log.Info("Shutting down kafka scraper")
	}()
}

type ConsumerHandler struct{}

func (h ConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h ConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h ConsumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		MsgChan <- msg.Value
		sess.MarkMessage(msg, "")
	}
	return nil
}
