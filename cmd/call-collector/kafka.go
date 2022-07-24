package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	. "github.com/dmkorzh/history-service/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func kafkaConfig() (kafka.ConfigMap, error) {
	cfg := make(kafka.ConfigMap)
	brokers := viper.GetStringSlice(KEY_BROKERS)
	if brokers == nil {
		return nil, fmt.Errorf("No kafka broker specified!")
	}
	_ = cfg.SetKey("bootstrap.servers", strings.Join(brokers, ","))
	_ = cfg.SetKey("session.timeout.ms", 6000)
	_ = cfg.SetKey("go.logs.channel.enable", true)
	_ = cfg.SetKey("compression.type", "lz4")
	return cfg, nil
}

func runKafkaWriter(msgC <-chan *kafka.Message, config *kafka.ConfigMap) (<-chan error, error) {
	var (
		wg     = &sync.WaitGroup{}
		abortC = make(chan struct{})
		errC   = make(chan error, 1)
	)
	producer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}
	logKafka(wg, producer)
	go serveEventsQueue(producer)

	go func() {
	MsgLoop:
		for msg := range msgC {
			for cnt := 0; ; cnt++ {
				if err := producer.Produce(msg, nil); err != nil {
					if e, ok := err.(kafka.Error); ok && e.IsFatal() {
						errC <- err
						goto Abort
					}
					if cnt == 0 {
						log.Warnf("error sending message: %s", err.Error())
					} else if cnt%10 == 0 {
						log.Warnf("error sending message (%d retries): %s", cnt, err.Error())
					}
					select {
					case <-time.After(5 * time.Second):
					case <-abortC:
						errC <- errors.New("aborting kafka producer, some messages have been lost forever")
						goto Abort
					}
				} else {
					continue MsgLoop
				}
			}
		}
		if n := producer.Flush(5000); err != nil {
			log.Warnf("failed to flush %d messages", n)
			errC <- fmt.Errorf("failed to flush %d messages", n)
		}
	Abort:
		producer.Close()
		wg.Wait()
		close(errC)
	}()
	return errC, nil
}

func serveEventsQueue(p *kafka.Producer) {
	for e := range p.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Warnf("Error delivering message: %s", ev.TopicPartition.Error)
			} else {
				log.Infof("Delivered message to %v\n", ev.TopicPartition)
			}
		case kafka.Error:
			log.Warnf("Kafka error: %s", ev)
		default:
			log.Warnf("Unknown event: %s", ev)
		}
	}
}

func logKafka(wg *sync.WaitGroup, producer *kafka.Producer) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for evt := range producer.Logs() {
			log.Infof("%s %s %s", evt.Name, evt.Tag, evt.Message)
		}
	}()
}
