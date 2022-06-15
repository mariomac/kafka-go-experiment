package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/paulbellamy/ratecounter"
	kafkago "github.com/segmentio/kafka-go"

	"golang.org/x/time/rate"
)

var topic = "test-topic"

func main() {
	batchSize := flag.Int("s", 50, "batch size")
	batchTimeout := flag.Duration("t", time.Second, "batch timeout")
	frequency := flag.Int("mps", 500, "messages per second")
	async := flag.Bool("a", false, "asynchronous writing")
	lib := flag.String("l", "segmentio", "producer lib: segmentio (kafka-go), sarama or confluent")
	brokerAddress := flag.String("b", "127.0.0.1:29092", "Kafka broker address")

	help := flag.Bool("h", false, "this help")
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}
	fmt.Printf("messages/sec = %d\tbatchSize = %d\tbatchTimeout = %v\n",
		*frequency, *batchSize, *batchTimeout)

	// periodically print how many messages have been sent
	submitRate := ratecounter.NewRateCounter(5 * time.Second)
	go func() {
		ticker := time.Tick(5 * time.Second)
		for {
			<-ticker
			log.Printf("%.1f messages/s",
				float64(submitRate.Rate())/5.0)
		}
	}()

	var kafkaWriter func([]byte) error
	switch *lib {
	case "sarama":
		cfg := sarama.NewConfig()
		cfg.Producer.Flush.Messages = *batchSize
		cfg.Producer.Flush.Frequency = *batchTimeout
		cfg.Version = sarama.V0_10_0_0
		if *async {
			ap, err := sarama.NewAsyncProducer([]string{*brokerAddress}, cfg)
			if err != nil {
				panic(err)
			}
			kafkaWriter = (&saramaAsyncWriter{writer: ap}).write
		} else {
			cfg.Producer.RequiredAcks = sarama.WaitForAll
			cfg.Producer.Return.Successes = true
			cfg.Producer.Return.Errors = true
			sp, err := sarama.NewSyncProducer([]string{*brokerAddress}, cfg)
			if err != nil {
				panic(err)
			}
			kafkaWriter = (&saramaWriter{writer: sp}).write
		}
	case "segmentio":
		writer := kafkago.Writer{
			Addr:                   kafkago.TCP(*brokerAddress),
			Topic:                  topic,
			BatchSize:              *batchSize,
			BatchTimeout:           *batchTimeout,
			AllowAutoTopicCreation: true,
			Async:                  *async,
		}
		kafkaWriter = (&kafkaGoWriter{writer: writer}).write
	case "confluent":
		acks := "all"
		if *async {
			acks = "0"
		}
		prod, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": *brokerAddress,
			"batch.size":        *batchSize,
			"linger.ms":         int(batchTimeout.Milliseconds()),
			"acks":              acks,
		})
		if err != nil {
			panic(err)
		}
		kafkaWriter = (&confluentWriter{writer: prod}).write
	}
	limit := rate.NewLimiter(rate.Limit(*frequency), 1)
	for {
		if err := limit.Wait(context.Background()); err != nil {
			panic(err)
		}
		if err := kafkaWriter([]byte(`{}`)); err != nil {
			log.Println("ERROR!", err)
		} else {
			submitRate.Incr(1)
		}
	}
}

type kafkaGoWriter struct {
	writer kafkago.Writer
}

func (kgw *kafkaGoWriter) write(msg []byte) error {
	return kgw.writer.WriteMessages(context.Background(), kafkago.Message{Value: msg})
}

type saramaWriter struct {
	writer sarama.SyncProducer
}

func (sw *saramaWriter) write(msg []byte) error {
	_, _, err := sw.writer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: -1,
		Value:     sarama.StringEncoder(msg),
	})
	return err
}

type saramaAsyncWriter struct {
	writer sarama.AsyncProducer
}

func (saw *saramaAsyncWriter) write(msg []byte) error {
	saw.writer.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Partition: -1,
		Value:     sarama.StringEncoder(msg),
	}
	return nil
}

type confluentWriter struct {
	writer *kafka.Producer
}

func (cw *confluentWriter) write(msg []byte) error {
	return cw.writer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          msg,
	}, nil)
}
