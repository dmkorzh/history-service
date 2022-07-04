package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/sirupsen/logrus"
)

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
