package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/dmkorzh/history-service/parser"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	apiRequestPath = "/call"
	apiRequestSize = 1024 * 1024
)

func (c *HttpCollector) handler(msgC chan<- *kafka.Message) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		addr := parseRemoteAddr(r)
	Check:
		for c.access != nil {
			for i := range c.access {
				if c.access[i].Contains(addr) {
					break Check
				}
			}
			log.Warnf("unexpected request from %s at %s", addr, r.RequestURI)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			log.Warnf("unexpected %s request", r.Method)
			return
		}
		if r.RequestURI != apiRequestPath {
			http.Error(w, "not found", http.StatusNotFound)
			log.Warnf("unexpected request at %s", r.RequestURI)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, apiRequestSize))
		if err != nil {
			http.Error(w, "error reading body", http.StatusInternalServerError)
			log.Warnf("error reading request: %s", err.Error())
			return
		}
		msg := &kafka.Message{
			Timestamp:     time.Now(),
			TimestampType: kafka.TimestampCreateTime,
			Headers: []kafka.Header{{
				Key:   "source",
				Value: []byte(r.RemoteAddr),
			}},
			TopicPartition: kafka.TopicPartition{
				Partition: kafka.PartitionAny,
			},
		}
		if key, err := getKey(body); err == nil {
			call, err := parser.ParseCall(body)
			if err != nil {
				http.Error(w, "failed to parse call", http.StatusInternalServerError)
				log.Warnf("failed to parse call: %s", err.Error())
			}
			msg.TopicPartition.Topic = &c.msgTopic
			msg.Key = key
			msg.Value = call
			msgC <- msg
			http.Error(w, "OK", http.StatusOK)
			return
		} else if c.failTopic != "" {
			msg.TopicPartition.Topic = &c.failTopic
			msg.Value = body
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "error",
				Value: []byte(err.Error()),
			})
			msgC <- msg
		}
		http.Error(w, "failed to process message", http.StatusInternalServerError)
	}
}

func parseRemoteAddr(r *http.Request) net.IP {
	if r.RemoteAddr == "" {
		return nil
	}
	if r.RemoteAddr[0] == '[' {
		if idx := strings.IndexByte(r.RemoteAddr[1:], ']'); idx != -1 {
			return net.ParseIP(r.RemoteAddr[1 : idx+1])
		} else {
			return nil
		}
	} else if idx := strings.IndexByte(r.RemoteAddr, ':'); idx != -1 {
		return net.ParseIP(r.RemoteAddr[:idx])
	} else {
		return net.ParseIP(r.RemoteAddr)
	}
}

func getKey(json []byte) ([]byte, error) {
	if !gjson.ValidBytes(json) {
		return nil, errors.New("invalid input JSON")
	}
	queueKey := gjson.GetBytes(json, "departmentID")
	if queueKey.Type != gjson.String || queueKey.Str == "" {
		return nil, errors.New("JSON has invalid department key")
	}
	return []byte(queueKey.Str), nil
}
