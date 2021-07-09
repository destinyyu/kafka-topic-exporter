package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

func main() {
	brokerString := flag.String("brokers", "", "Kafka brokers")
	topics := flag.String("topics", "", "consumer topics")
	group := flag.String("group", "", "consumer group")
	port := flag.Int("port", 9000, "Port to export metrics on")
	path := flag.String("path", "/metrics", "Path to export metrics on")
	expiration := flag.Int64("expiration", 0, "expired seconds for metrics, default 0 means never expiration")
	exportTimestamp := flag.Bool("export_timestamp", false, "export timestamp from custom data, default use current time")
	droplabels := flag.String("drop_labels", "", "labels which will be dropped")
	level := flag.String("level", "info", "Logger level")
	flag.Parse()

	if *brokerString == "" || *topics == "" {
		panic("ERROR: broker or topic is none")
	}

	setupLogger(*level)

	kafka := newKafka(*brokerString, *group)
	defer func() {
		if err := kafka.Close(); err != nil {
			panic(err)
		}
	}()

	serverConfig := newServerConfig(*port, *path, *expiration, *exportTimestamp)
	scrapeConfig := newScrapeConfig(*topics, *droplabels)

	enforceGracefulShutdown(func(wg *sync.WaitGroup, shutdown chan struct{}) {
		startKafkaScraper(wg, shutdown, kafka, scrapeConfig)
		generateMetrics(wg, shutdown, serverConfig)
		startMetricsServer(wg, shutdown, serverConfig)
	})
}

func enforceGracefulShutdown(f func(wg *sync.WaitGroup, shutdown chan struct{})) {
	wg := &sync.WaitGroup{}
	shutdown := make(chan struct{})
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-signals
		close(shutdown)
	}()

	log.Info("Graceful shutdown enabled")
	f(wg, shutdown)

	<-shutdown
	wg.Wait()
}

func newKafka(brokerString, group string) sarama.ConsumerGroup {
	brokers := strings.Split(brokerString, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
		if !strings.ContainsRune(brokers[i], ':') {
			brokers[i] += ":9092"
		}
	}
	log.WithField("brokers", brokers).Info("connecting to kafka cluster")

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_3_0_0
	cfg.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(brokers, group, cfg)
	if err != nil {
		log.Fatal(err)
	}

	// track errors
	go func() {
		for err := range consumer.Errors() {
			log.Error(err)
		}
	}()

	return consumer
}

func setupLogger(level string) {
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.JSONFormatter{})
}
